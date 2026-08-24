package block

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/alkaid/miniprometheus/internal/encode"
	"github.com/alkaid/miniprometheus/internal/head"
	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/model"
)

type SeriesChunks struct {
	Labels model.Labels
	Chunks [][]byte
	MinT   int64
	MaxT   int64
}

type Block struct {
	Dir    string
	Meta   Meta
	series []SeriesChunks
	idx    *index.Inverted
	ids    []model.SeriesID
}

func Persist(root string, frozen []head.FrozenSeries) (*Block, error) {
	if len(frozen) == 0 {
		return nil, nil
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	b := &Block{Dir: dir, idx: index.NewInverted(), series: make([]SeriesChunks, 0, len(frozen))}
	var minT int64 = 1 << 62
	var maxT int64
	samples, chunks := 0, 0
	for _, fs := range frozen {
		if fs.MinT < minT {
			minT = fs.MinT
		}
		if fs.MaxT > maxT {
			maxT = fs.MaxT
		}
		sid, _ := b.idx.GetOrCreate(fs.Labels)
		b.ids = append(b.ids, sid)
		b.series = append(b.series, SeriesChunks{Labels: fs.Labels, Chunks: fs.Chunks, MinT: fs.MinT, MaxT: fs.MaxT})
		chunks += len(fs.Chunks)
		for _, c := range fs.Chunks {
			if ss, err := encode.Decode(c); err == nil {
				samples += len(ss)
			}
		}
	}
	if err := writeChunks(filepath.Join(dir, "chunks.bin"), b.series); err != nil {
		return nil, err
	}
	_ = WriteSparseIndex(dir, b.series)
	b.Meta = Meta{ULID: id, MinT: minT, MaxT: maxT}
	b.Meta.Stats.NumSeries = len(b.series)
	b.Meta.Stats.NumSamples = samples
	b.Meta.Stats.NumChunks = chunks
	raw, _ := json.MarshalIndent(b.Meta, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), raw, 0o644); err != nil {
		return nil, err
	}
	return b, nil
}

func writeChunks(path string, ss []SeriesChunks) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(ss)))
	if _, err := f.Write(hdr[:]); err != nil {
		return err
	}
	for _, s := range ss {
		lb, _ := json.Marshal(s.Labels)
		binary.BigEndian.PutUint32(hdr[:], uint32(len(lb)))
		_, _ = f.Write(hdr[:])
		_, _ = f.Write(lb)
		var tbuf [16]byte
		binary.BigEndian.PutUint64(tbuf[0:8], uint64(s.MinT))
		binary.BigEndian.PutUint64(tbuf[8:16], uint64(s.MaxT))
		_, _ = f.Write(tbuf[:])
		binary.BigEndian.PutUint32(hdr[:], uint32(len(s.Chunks)))
		_, _ = f.Write(hdr[:])
		for _, c := range s.Chunks {
			binary.BigEndian.PutUint32(hdr[:], uint32(len(c)))
			_, _ = f.Write(hdr[:])
			_, _ = f.Write(c)
		}
	}
	return nil
}

func Load(dir string) (*Block, error) {
	metaRaw, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return nil, err
	}
	var meta Meta
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(dir, "chunks.bin"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var hdr [4]byte
	if _, err := f.Read(hdr[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint32(hdr[:]))
	b := &Block{Dir: dir, Meta: meta, idx: index.NewInverted()}
	for i := 0; i < n; i++ {
		if _, err := f.Read(hdr[:]); err != nil {
			return nil, err
		}
		lb := make([]byte, binary.BigEndian.Uint32(hdr[:]))
		if _, err := f.Read(lb); err != nil {
			return nil, err
		}
		var ls model.Labels
		if err := json.Unmarshal(lb, &ls); err != nil {
			return nil, err
		}
		var tbuf [16]byte
		if _, err := f.Read(tbuf[:]); err != nil {
			return nil, err
		}
		minT := int64(binary.BigEndian.Uint64(tbuf[0:8]))
		maxT := int64(binary.BigEndian.Uint64(tbuf[8:16]))
		if _, err := f.Read(hdr[:]); err != nil {
			return nil, err
		}
		nc := int(binary.BigEndian.Uint32(hdr[:]))
		chunks := make([][]byte, 0, nc)
		for j := 0; j < nc; j++ {
			if _, err := f.Read(hdr[:]); err != nil {
				return nil, err
			}
			cb := make([]byte, binary.BigEndian.Uint32(hdr[:]))
			if _, err := f.Read(cb); err != nil {
				return nil, err
			}
			chunks = append(chunks, cb)
		}
		sid, _ := b.idx.GetOrCreate(ls)
		b.ids = append(b.ids, sid)
		b.series = append(b.series, SeriesChunks{Labels: ls, Chunks: chunks, MinT: minT, MaxT: maxT})
	}
	return b, nil
}

func (b *Block) Query(ms []*index.Matcher, mint, maxt int64) []model.TimeSeries {
	if maxt < b.Meta.MinT || mint > b.Meta.MaxT {
		return nil
	}
	bm, _ := b.idx.Lookup(ms)
	ids := index.IDsFromBitmap(bm)
	out := make([]model.TimeSeries, 0, len(ids))
	idset := make(map[model.SeriesID]int, len(b.ids))
	for i, id := range b.ids {
		idset[id] = i
	}
	for _, id := range ids {
		i, ok := idset[id]
		if !ok {
			continue
		}
		s := b.series[i]
		var pts []model.Sample
		for _, c := range s.Chunks {
			ss, err := encode.Decode(c)
			if err != nil {
				continue
			}
			for _, p := range ss {
				if p.T >= mint && p.T <= maxt {
					pts = append(pts, p)
				}
			}
		}
		if len(pts) > 0 {
			out = append(out, model.TimeSeries{Labels: s.Labels, Samples: pts})
		}
	}
	return out
}

type Store struct {
	mu     sync.RWMutex
	dir    string
	blocks []*Block
}

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		b, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			logger.L().Warn("skip block", "dir", e.Name(), "err", err.Error())
			continue
		}
		s.blocks = append(s.blocks, b)
	}
	return s, nil
}

func (s *Store) Add(b *Block) {
	if b == nil {
		return
	}
	s.mu.Lock()
	s.blocks = append(s.blocks, b)
	s.mu.Unlock()
}

func (s *Store) Query(ms []*index.Matcher, mint, maxt int64) []model.TimeSeries {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var acc []model.TimeSeries
	for _, b := range s.blocks {
		acc = append(acc, b.Query(ms, mint, maxt)...)
	}
	return acc
}

func (s *Store) Expire(before int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.blocks[:0]
	for _, b := range s.blocks {
		if b.Meta.MaxT < before {
			_ = os.RemoveAll(b.Dir)
			continue
		}
		kept = append(kept, b)
	}
	s.blocks = kept
}

func (s *Store) List() []Meta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Meta, 0, len(s.blocks))
	for _, b := range s.blocks {
		out = append(out, b.Meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MinT < out[j].MinT })
	return out
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blocks)
}
