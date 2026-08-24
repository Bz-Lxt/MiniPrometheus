package head

import (
	"sync"

	"github.com/alkaid/miniprometheus/internal/encode"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/ring"
)

type memSeries struct {
	mu      sync.Mutex
	id      model.SeriesID
	labels  model.Labels
	pending []model.Sample
	sealed  [][]byte
	ring    *ring.Buffer
	minT    int64
	maxT    int64
	points  int
	rawB    int
	compB   int
}

func newMemSeries(id model.SeriesID, ls model.Labels) *memSeries {
	return &memSeries{
		id:     id,
		labels: ls,
		ring:   ring.New(16),
		minT:   1 << 62,
	}
}

func (m *memSeries) append(s model.Sample) encode.Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.pending) > 0 && s.T <= m.pending[len(m.pending)-1].T {
		m.pending[len(m.pending)-1] = s
		return encode.Stats{}
	}
	if encode.ShouldCut(m.pending, s.T) {
		m.sealLocked()
	}
	m.pending = append(m.pending, s)
	if s.T < m.minT {
		m.minT = s.T
	}
	if s.T > m.maxT {
		m.maxT = s.T
	}
	m.points++
	if len(m.pending) >= encode.MaxPoints {
		return m.sealLocked()
	}
	return encode.Stats{}
}

func (m *memSeries) sealLocked() encode.Stats {
	if len(m.pending) == 0 {
		return encode.Stats{}
	}
	blob, st, err := encode.Encode(m.pending)
	if err == nil {
		m.sealed = append(m.sealed, blob)
		m.ring.AppendChunk(m.pending)
		m.rawB += st.RawBytes
		m.compB += st.CompBytes
	}
	m.pending = m.pending[:0]
	return st
}

func (m *memSeries) flush() encode.Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sealLocked()
}

func (m *memSeries) read(mint, maxt int64) []model.Sample {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []model.Sample
	for _, blob := range m.sealed {
		ss, err := encode.Decode(blob)
		if err != nil {
			continue
		}
		for _, p := range ss {
			if p.T >= mint && p.T <= maxt {
				out = append(out, p)
			}
		}
	}
	for _, p := range m.pending {
		if p.T >= mint && p.T <= maxt {
			out = append(out, p)
		}
	}
	return out
}

func (m *memSeries) snapshotPending() []model.Sample {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]model.Sample(nil), m.pending...)
}
