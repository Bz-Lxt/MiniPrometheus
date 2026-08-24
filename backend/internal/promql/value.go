package promql

import (
	"math"
	"strconv"

	"github.com/alkaid/miniprometheus/internal/model"
)

type ValueKind int

const (
	ValNone ValueKind = iota
	ValScalar
	ValVector
	ValMatrix
)

type SamplePoint struct {
	T int64
	V float64
}

type Series struct {
	Labels model.Labels
	Points []SamplePoint
}

type Value struct {
	Kind   ValueKind
	Scalar SamplePoint
	Series []Series
}

func labelsKey(ls model.Labels) string {
	return ls.String()
}

func toInstant(ss []Series, t int64) []Series {
	out := make([]Series, 0, len(ss))
	for _, s := range ss {
		if len(s.Points) == 0 {
			continue
		}
		p := s.Points[len(s.Points)-1]
		if t > 0 {
			p.T = t
		}
		out = append(out, Series{Labels: s.Labels, Points: []SamplePoint{p}})
	}
	return out
}

func formatPair(t int64, v float64) []any {
	return []any{float64(t) / 1000.0, formatV(v)}
}

func formatV(v float64) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
}

func PromQLResult(v Value, resultType string) map[string]any {
	switch v.Kind {
	case ValScalar:
		return map[string]any{
			"resultType": "scalar",
			"result":     formatPair(v.Scalar.T, v.Scalar.V),
		}
	case ValVector:
		res := make([]map[string]any, 0, len(v.Series))
		for _, s := range v.Series {
			if len(s.Points) == 0 {
				continue
			}
			p := s.Points[len(s.Points)-1]
			res = append(res, map[string]any{
				"metric": s.Labels.Map(),
				"value":  formatPair(p.T, p.V),
			})
		}
		return map[string]any{"resultType": "vector", "result": res}
	default:
		res := make([]map[string]any, 0, len(v.Series))
		for _, s := range v.Series {
			vals := make([][]any, 0, len(s.Points))
			for _, p := range s.Points {
				vals = append(vals, formatPair(p.T, p.V))
			}
			res = append(res, map[string]any{
				"metric": s.Labels.Map(),
				"values": vals,
			})
		}
		if resultType == "" {
			resultType = "matrix"
		}
		return map[string]any{"resultType": resultType, "result": res}
	}
}
