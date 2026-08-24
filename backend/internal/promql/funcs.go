package promql

import (
	"math"
	"sort"
	"strconv"

	"github.com/alkaid/miniprometheus/internal/model"
)

func evalCall(name string, in Value, param float64) Value {
	switch name {
	case "rate":
		return overTime(in, rateFn)
	case "irate":
		return overTime(in, irateFn)
	case "increase":
		return overTime(in, increaseFn)
	case "delta":
		return overTime(in, deltaFn)
	case "avg_over_time":
		return overTime(in, avgFn)
	case "max_over_time":
		return overTime(in, maxFn)
	case "min_over_time":
		return overTime(in, minFn)
	case "sum_over_time":
		return overTime(in, sumFn)
	case "count_over_time":
		return overTime(in, countFn)
	case "histogram_quantile":
		return histogramQuantile(in, param)
	default:
		return in
	}
}

type reduceFn func(pts []SamplePoint) (float64, bool)

func overTime(in Value, fn reduceFn) Value {
	out := Value{Kind: ValVector}
	for _, s := range in.Series {
		v, ok := fn(s.Points)
		if !ok {
			continue
		}
		t := s.Points[len(s.Points)-1].T
		out.Series = append(out.Series, Series{
			Labels: s.Labels.Without(model.MetricName),
			Points: []SamplePoint{{T: t, V: v}},
		})
	}
	return out
}

func rateFn(pts []SamplePoint) (float64, bool) {
	inc, ok := increaseFn(pts)
	if !ok {
		return 0, false
	}
	dt := float64(pts[len(pts)-1].T-pts[0].T) / 1000.0
	if dt <= 0 {
		return 0, false
	}
	return inc / dt, true
}

func irateFn(pts []SamplePoint) (float64, bool) {
	if len(pts) < 2 {
		return 0, false
	}
	a, b := pts[len(pts)-2], pts[len(pts)-1]
	dv := b.V - a.V
	if dv < 0 {
		dv = b.V
	}
	dt := float64(b.T-a.T) / 1000.0
	if dt <= 0 {
		return 0, false
	}
	return dv / dt, true
}

func increaseFn(pts []SamplePoint) (float64, bool) {
	if len(pts) < 2 {
		return 0, false
	}
	var sum float64
	prev := pts[0].V
	for i := 1; i < len(pts); i++ {
		v := pts[i].V
		if v < prev {
			sum += v
		} else {
			sum += v - prev
		}
		prev = v
	}
	return sum, true
}

func deltaFn(pts []SamplePoint) (float64, bool) {
	if len(pts) < 2 {
		return 0, false
	}
	return pts[len(pts)-1].V - pts[0].V, true
}

func avgFn(pts []SamplePoint) (float64, bool) {
	if len(pts) == 0 {
		return 0, false
	}
	var s float64
	for _, p := range pts {
		s += p.V
	}
	return s / float64(len(pts)), true
}

func maxFn(pts []SamplePoint) (float64, bool) {
	if len(pts) == 0 {
		return 0, false
	}
	m := pts[0].V
	for _, p := range pts[1:] {
		if p.V > m {
			m = p.V
		}
	}
	return m, true
}

func minFn(pts []SamplePoint) (float64, bool) {
	if len(pts) == 0 {
		return 0, false
	}
	m := pts[0].V
	for _, p := range pts[1:] {
		if p.V < m {
			m = p.V
		}
	}
	return m, true
}

func sumFn(pts []SamplePoint) (float64, bool) {
	if len(pts) == 0 {
		return 0, false
	}
	var s float64
	for _, p := range pts {
		s += p.V
	}
	return s, true
}

func countFn(pts []SamplePoint) (float64, bool) {
	return float64(len(pts)), len(pts) > 0
}

func histogramQuantile(in Value, q float64) Value {
	groups := map[string][]Series{}
	for _, s := range in.Series {
		key := s.Labels.Without("le", model.MetricName).String()
		groups[key] = append(groups[key], s)
	}
	out := Value{Kind: ValVector}
	for _, gs := range groups {
		type bucket struct {
			le float64
			v  float64
			t  int64
			ls model.Labels
		}
		var bs []bucket
		for _, s := range gs {
			le := s.Labels.Get("le")
			if le == "" || len(s.Points) == 0 {
				continue
			}
			fv, err := strconv.ParseFloat(le, 64)
			if err != nil {
				if le == "+Inf" {
					fv = math.Inf(1)
				} else {
					continue
				}
			}
			p := s.Points[len(s.Points)-1]
			bs = append(bs, bucket{le: fv, v: p.V, t: p.T, ls: s.Labels})
		}
		if len(bs) == 0 {
			continue
		}
		sort.Slice(bs, func(i, j int) bool { return bs[i].le < bs[j].le })
		total := bs[len(bs)-1].v
		if total <= 0 || q < 0 || q > 1 {
			continue
		}
		rank := q * total
		var prevLe, prevV float64
		val := math.NaN()
		for i, b := range bs {
			if b.v >= rank {
				if i == 0 || math.IsInf(b.le, 1) {
					val = b.le
				} else {
					den := b.v - prevV
					if den <= 0 {
						val = b.le
					} else {
						val = prevLe + (b.le-prevLe)*(rank-prevV)/den
					}
				}
				break
			}
			prevLe, prevV = b.le, b.v
		}
		out.Series = append(out.Series, Series{
			Labels: bs[0].ls.Without("le", model.MetricName),
			Points: []SamplePoint{{T: bs[0].t, V: val}},
		})
	}
	return out
}
