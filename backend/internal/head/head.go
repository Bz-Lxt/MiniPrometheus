package head

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/wal"
)

type Head struct {
	mu       sync.RWMutex
	idx      *index.Inverted
	series   map[model.SeriesID]*memSeries
	wal      *wal.WAL
	samples  atomic.Int64
	bytesIn  atomic.Int64
	bytesOut atomic.Int64
	dropped  atomic.Int64
}

func New(w *wal.WAL) *Head {
	return &Head{
		idx:    index.NewInverted(),
		series: make(map[model.SeriesID]*memSeries),
		wal:    w,
	}
}

func (h *Head) Index() *index.Inverted { return h.idx }

func (h *Head) Append(ls model.Labels, t int64, v float64) (model.SeriesID, error) {
	ls = model.Normalize(ls)
	id, created := h.idx.GetOrCreate(ls)
	h.mu.Lock()
	ms, ok := h.series[id]
	if !ok {
		ms = newMemSeries(id, ls)
		h.series[id] = ms
	}
	h.mu.Unlock()
	if h.wal != nil {
		if created {
			if err := h.wal.LogSeries(uint32(id), ls); err != nil {
				return id, err
			}
		}
		if err := h.wal.LogSample(uint32(id), t, v); err != nil {
			return id, err
		}
	}
	st := ms.append(model.Sample{T: t, V: v})
	h.samples.Add(1)
	h.bytesIn.Add(16)
	if st.CompBytes > 0 {
		h.bytesOut.Add(int64(st.CompBytes))
	}
	return id, nil
}

func (h *Head) AppendMany(ls model.Labels, ss []model.Sample) error {
	for _, s := range ss {
		if _, err := h.Append(ls, s.T, s.V); err != nil {
			return err
		}
	}
	return nil
}

func (h *Head) Query(ms []*index.Matcher, mint, maxt int64) ([]model.TimeSeries, index.LookupStat) {
	bm, st := h.idx.Lookup(ms)
	ids := index.IDsFromBitmap(bm)
	out := make([]model.TimeSeries, 0, len(ids))
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, id := range ids {
		msy, ok := h.series[id]
		if !ok {
			continue
		}
		ls, _ := h.idx.Labels(id)
		pts := msy.read(mint, maxt)
		if len(pts) == 0 {
			continue
		}
		out = append(out, model.TimeSeries{Labels: ls, Samples: pts})
	}
	return out, st
}

func (h *Head) SeriesMeta(ms []*index.Matcher) []model.Labels {
	bm, _ := h.idx.Lookup(ms)
	ids := index.IDsFromBitmap(bm)
	out := make([]model.Labels, 0, len(ids))
	for _, id := range ids {
		if ls, ok := h.idx.Labels(id); ok {
			out = append(out, ls)
		}
	}
	return out
}

func (h *Head) FlushAll() {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, s := range h.series {
		st := s.flush()
		if st.CompBytes > 0 {
			h.bytesOut.Add(int64(st.CompBytes))
		}
	}
}

func (h *Head) SnapshotSealed() []FrozenSeries {
	h.FlushAll()
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]FrozenSeries, 0, len(h.series))
	for _, s := range h.series {
		s.mu.Lock()
		if len(s.sealed) == 0 {
			s.mu.Unlock()
			continue
		}
		blobs := append([][]byte(nil), s.sealed...)
		ls := s.labels
		minT, maxT := s.minT, s.maxT
		s.mu.Unlock()
		out = append(out, FrozenSeries{ID: s.id, Labels: ls, Chunks: blobs, MinT: minT, MaxT: maxT})
	}
	return out
}

func (h *Head) DropSealed(ids []model.SeriesID) {
	set := make(map[model.SeriesID]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for id := range set {
		if s, ok := h.series[id]; ok {
			s.mu.Lock()
			s.sealed = s.sealed[:0]
			s.mu.Unlock()
		}
	}
}

type FrozenSeries struct {
	ID     model.SeriesID
	Labels model.Labels
	Chunks [][]byte
	MinT   int64
	MaxT   int64
}

type Stats struct {
	Series     int   `json:"series"`
	Samples    int64 `json:"samples"`
	BytesIn    int64 `json:"bytes_in"`
	BytesOut   int64 `json:"bytes_out"`
	BPS        float64 `json:"bytes_per_sample"`
	Dropped    int64 `json:"dropped"`
	UpdatedAt  string `json:"updated_at"`
}

func (h *Head) Stats() Stats {
	in := h.bytesIn.Load()
	out := h.bytesOut.Load()
	n := h.samples.Load()
	bps := 0.0
	if n > 0 && out > 0 {
		bps = float64(out) / float64(n)
	}
	return Stats{
		Series:    h.idx.SeriesCount(),
		Samples:   n,
		BytesIn:   in,
		BytesOut:  out,
		BPS:       bps,
		Dropped:   h.dropped.Load(),
		UpdatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
}

func (h *Head) RestoreSeries(id uint32, ls model.Labels) {
	sid := model.SeriesID(id)
	h.idx.Restore(sid, ls)
	h.mu.Lock()
	if _, ok := h.series[sid]; !ok {
		h.series[sid] = newMemSeries(sid, model.Normalize(ls))
	}
	h.mu.Unlock()
}

func (h *Head) RestoreSample(id uint32, t int64, v float64) {
	h.mu.RLock()
	ms := h.series[model.SeriesID(id)]
	h.mu.RUnlock()
	if ms == nil {
		return
	}
	ms.append(model.Sample{T: t, V: v})
	h.samples.Add(1)
	h.bytesIn.Add(16)
}

func (h *Head) WAL() *wal.WAL { return h.wal }
