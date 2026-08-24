package wal

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/alkaid/miniprometheus/internal/logger"
)

func RepairTail(dir string) (int, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var files []string
	for _, e := range ents {
		if filepath.Ext(e.Name()) == ".wal" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	if len(files) == 0 {
		return 0, nil
	}
	sort.Strings(files)
	path := files[len(files)-1]
	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var good int64
	dropped := 0
	for {
		pos, _ := f.Seek(0, io.SeekCurrent)
		var hdr [4]byte
		if _, err := io.ReadFull(f, hdr[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return dropped, err
		}
		ln := int(binary.BigEndian.Uint32(hdr[:]))
		if ln <= 0 || ln > 16<<20 {
			dropped++
			break
		}
		rec := make([]byte, ln)
		if _, err := io.ReadFull(f, rec); err != nil {
			dropped++
			break
		}
		if _, _, ok := decodePayload(rec); !ok {
			dropped++
			break
		}
		good = pos + 4 + int64(ln)
	}
	st, err := f.Stat()
	if err != nil {
		return dropped, err
	}
	if good < st.Size() {
		logger.L().Warn("wal truncate dirty tail", "file", path, "from", st.Size(), "to", good)
		if err := f.Truncate(good); err != nil {
			return dropped, err
		}
	}
	return dropped, nil
}
