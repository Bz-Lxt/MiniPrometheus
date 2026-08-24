package promql

import (
	"math"
	"sort"

	"github.com/alkaid/miniprometheus/internal/model"
)

func evalAgg(op string, in Value, by, without []string, param float64, hasParam bool) Value {
	type grp struct {
		ls model.Labels
		vs []float64
		t  int64
	}
	groups := map[string]*grp{}
	for _, s := range in.Series {
		if len(s.Points) == 0 {
			continue
		}
		var keyLs model.Labels
		switch {
		case len(by) > 0:
			keyLs = s.Labels.Keep(by...)
		case len(without) > 0:
			keyLs = s.Labels.Without(append(without, model.MetricName)...)
		default:
			keyLs = nil
		}
		k := keyLs.String()
		g, ok := groups[k]
		if !ok {
			g = &grp{ls: keyLs}
			groups[k] = g
		}
		p := s.Points[len(s.Points)-1]
		g.vs = append(g.vs, p.V)
		g.t = p.T
	}
	out := Value{Kind: ValVector}
	for _, g := range groups {
		var v float64
		switch op {
		case "sum":
			for _, x := range g.vs {
				v += x
			}
		case "avg":
			for _, x := range g.vs {
				v += x
			}
			if len(g.vs) > 0 {
				v /= float64(len(g.vs))
			}
		case "min":
			v = math.Inf(1)
			for _, x := range g.vs {
				if x < v {
					v = x
				}
			}
		case "max":
			v = math.Inf(-1)
			for _, x := range g.vs {
				if x > v {
					v = x
				}
			}
		case "count":
			v = float64(len(g.vs))
		case "topk":
			return topk(in, int(param))
		case "quantile":
			v = quantile(g.vs, param)
		}
		if op == "topk" {
			continue
		}
		out.Series = append(out.Series, Series{Labels: g.ls, Points: []SamplePoint{{T: g.t, V: v}}})
	}
	_ = hasParam
	return out
}

func topk(in Value, k int) Value {
	if k <= 0 {
		return Value{Kind: ValVector}
	}
	type pair struct {
		s Series
		v float64
	}
	var ps []pair
	for _, s := range in.Series {
		if len(s.Points) == 0 {
			continue
		}
		ps = append(ps, pair{s: s, v: s.Points[len(s.Points)-1].V})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].v > ps[j].v })
	if k > len(ps) {
		k = len(ps)
	}
	out := Value{Kind: ValVector}
	for i := 0; i < k; i++ {
		out.Series = append(out.Series, ps[i].s)
	}
	return out
}

func quantile(vs []float64, q float64) float64 {
	if len(vs) == 0 || q < 0 || q > 1 {
		return math.NaN()
	}
	cp := append([]float64(nil), vs...)
	sort.Float64s(cp)
	if q == 0 {
		return cp[0]
	}
	if q == 1 {
		return cp[len(cp)-1]
	}
	pos := q * float64(len(cp)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return cp[lo]
	}
	return cp[lo] + (cp[hi]-cp[lo])*(pos-float64(lo))
}
