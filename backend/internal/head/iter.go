package head

import "github.com/alkaid/miniprometheus/internal/model"

func (h *Head) ForEach(fn func(id model.SeriesID, ls model.Labels)) {
	ids := h.idx.AllIDs()
	for _, id := range ids {
		if ls, ok := h.idx.Labels(id); ok {
			fn(id, ls)
		}
	}
}

func (h *Head) MinMaxTime() (int64, int64) {
	var minT int64 = 1 << 62
	var maxT int64
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, s := range h.series {
		s.mu.Lock()
		if s.minT < minT {
			minT = s.minT
		}
		if s.maxT > maxT {
			maxT = s.maxT
		}
		s.mu.Unlock()
	}
	if maxT == 0 {
		return 0, 0
	}
	return minT, maxT
}
