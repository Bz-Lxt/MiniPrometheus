package model

type SeriesID uint32

type SeriesRef struct {
	ID     SeriesID `json:"id"`
	Labels Labels   `json:"labels"`
	Hash   uint64   `json:"hash"`
}

type TimeSeries struct {
	Labels  Labels   `json:"metric"`
	Samples []Sample `json:"values,omitempty"`
}

type InstantSeries struct {
	Labels Labels  `json:"metric"`
	T      int64   `json:"t"`
	V      float64 `json:"v"`
}
