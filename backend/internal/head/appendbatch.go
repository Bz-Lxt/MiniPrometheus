package head

import (
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/remote"
)

func (h *Head) Ingest(req remote.WriteRequest) (int, error) {
	n := 0
	for _, s := range req.Series {
		if err := h.AppendMany(s.Labels, s.Samples); err != nil {
			return n, err
		}
		n += len(s.Samples)
	}
	return n, nil
}

func (h *Head) IngestOne(metric string, labs map[string]string, t int64, v float64) error {
	_, err := h.Append(model.FromMap(metric, labs), t, v)
	return err
}
