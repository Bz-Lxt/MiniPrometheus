package model

type Sample struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

type Point struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

func SamplesToPairs(ss []Sample) [][]any {
	out := make([][]any, len(ss))
	for i, s := range ss {
		out[i] = []any{float64(s.T) / 1000.0, formatFloat(s.V)}
	}
	return out
}

func formatFloat(v float64) string {
	return strconvFormat(v)
}
