package promql

type ProfileNode struct {
	ID         int           `json:"id"`
	Op         string        `json:"op"`
	Detail     string        `json:"detail"`
	DurationMS float64       `json:"duration_ms"`
	InSeries   int           `json:"in_series"`
	OutSeries  int           `json:"out_series"`
	Samples    int           `json:"samples_scanned"`
	HitIndex   bool          `json:"hit_index"`
	Children   []ProfileNode `json:"children"`
}

type Querier interface {
	Select(matchers []*SelectorHint, mint, maxt int64) ([]SeriesData, SelectStat)
}

type SelectorHint struct {
	Matchers any
	Range    int64
}

type SeriesData struct {
	Labels map[string]string
	Points [][2]float64
}

type SelectStat struct {
	DurationUS int64
	Series     int
	Samples    int
	HitIndex   bool
}
