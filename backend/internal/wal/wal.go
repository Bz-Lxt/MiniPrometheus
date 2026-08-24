package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/model"
)

const segSize = 32 << 20

type WAL struct {
	dir    string
	mu     sync.Mutex
	f      *os.File
	size   int
	closed sync.Once
	seq    int
}

func Open(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	w := &WAL{dir: dir}
	if err := w.rotate(false); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *WAL) rotate(force bool) error {
	entries, _ := os.ReadDir(w.dir)
	max := -1
	for _, e := range entries {
		var n int
		if _, err := fmt.Sscanf(e.Name(), "%08d.wal", &n); err == nil && n > max {
			max = n
		}
	}
	if w.f != nil {
		_ = w.f.Sync()
		_ = w.f.Close()
		w.f = nil
	}
	if !force && max >= 0 && w.seq == 0 {
		w.seq = max
	} else {
		w.seq = max + 1
		if w.seq < 0 {
			w.seq = 0
		}
	}
	path := filepath.Join(w.dir, fmt.Sprintf("%08d.wal", w.seq))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	w.f = f
	w.size = int(st.Size())
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return nil
}

func (w *WAL) append(rec []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return fmt.Errorf("wal closed")
	}
	if w.size+len(rec)+4 >= segSize {
		if err := w.rotate(true); err != nil {
			return err
		}
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(rec)))
	if _, err := w.f.Write(hdr[:]); err != nil {
		return err
	}
	if _, err := w.f.Write(rec); err != nil {
		return err
	}
	w.size += 4 + len(rec)
	return nil
}

func (w *WAL) LogSeries(id uint32, ls model.Labels) error {
	return w.append(encodeSeries(id, ls))
}

func (w *WAL) LogSample(id uint32, t int64, v float64) error {
	return w.append(encodeSample(id, t, v))
}

func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	return w.f.Sync()
}

func (w *WAL) Close() error {
	var err error
	w.closed.Do(func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.f != nil {
			err = w.f.Sync()
			_ = w.f.Close()
			w.f = nil
		}
	})
	return err
}

type Replayer interface {
	RestoreSeries(id uint32, ls model.Labels)
	RestoreSample(id uint32, t int64, v float64)
}

func Replay(dir string, dst Replayer) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var files []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".wal" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	n := 0
	for _, path := range files {
		c, err := replayFile(path, dst)
		n += c
		if err != nil {
			logger.L().Warn("wal replay truncated", "file", path, "err", err.Error(), "records", c)
			break
		}
	}
	return n, nil
}

func replayFile(path string, dst Replayer) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n := 0
	for {
		var hdr [4]byte
		if _, err := io.ReadFull(f, hdr[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return n, nil
			}
			return n, err
		}
		ln := int(binary.BigEndian.Uint32(hdr[:]))
		if ln <= 0 || ln > 16<<20 {
			return n, fmt.Errorf("invalid record length %d", ln)
		}
		rec := make([]byte, ln)
		if _, err := io.ReadFull(f, rec); err != nil {
			return n, err
		}
		typ, payload, ok := decodePayload(rec)
		if !ok {
			return n, fmt.Errorf("crc mismatch, skip tail")
		}
		switch typ {
		case RecSeries:
			id, ls, ok := decodeSeries(payload)
			if ok {
				dst.RestoreSeries(id, ls)
			}
		case RecSample:
			id, t, v, ok := decodeSample(payload)
			if ok {
				dst.RestoreSample(id, t, v)
				n++
			}
		}
	}
}
