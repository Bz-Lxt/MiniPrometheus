package api

import (
	"sort"

	"github.com/alkaid/miniprometheus/internal/model"
)

func mergeSeries(parts [][]model.TimeSeries) []model.TimeSeries {
	type acc struct {
		ls model.Labels
		m  map[int64]float64
	}
	idx := map[string]*acc{}
	order := []string{}
	for _, part := range parts {
		for _, s := range part {
			k := s.Labels.String()
			a, ok := idx[k]
			if !ok {
				a = &acc{ls: s.Labels, m: map[int64]float64{}}
				idx[k] = a
				order = append(order, k)
			}
			for _, p := range s.Samples {
				a.m[p.T] = p.V
			}
		}
	}
	out := make([]model.TimeSeries, 0, len(order))
	for _, k := range order {
		a := idx[k]
		ts := make([]int64, 0, len(a.m))
		for t := range a.m {
			ts = append(ts, t)
		}
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		ss := make([]model.Sample, 0, len(ts))
		for _, t := range ts {
			ss = append(ss, model.Sample{T: t, V: a.m[t]})
		}
		out = append(out, model.TimeSeries{Labels: a.ls, Samples: ss})
	}
	return out
}

func mergeStringSets(parts [][]string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range parts {
		for _, s := range p {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
