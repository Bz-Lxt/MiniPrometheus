package promql

import (
	"fmt"
	"sync"

	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/model"
)

func cacheSource(src SeriesSource, mint, maxt int64) SeriesSource {
	var mu sync.Mutex
	type pack struct {
		ss []model.TimeSeries
		st index.LookupStat
	}
	m := map[string]pack{}
	return func(ms []*index.Matcher, _, _ int64) ([]model.TimeSeries, index.LookupStat) {
		key := matcherKey(ms)
		mu.Lock()
		if p, ok := m[key]; ok {
			mu.Unlock()
			return filterTime(p.ss, mint, maxt), p.st
		}
		mu.Unlock()
		ss, st := src(ms, mint, maxt)
		mu.Lock()
		m[key] = pack{ss: ss, st: st}
		mu.Unlock()
		return filterTime(ss, mint, maxt), st
	}
}

func matcherKey(ms []*index.Matcher) string {
	var b string
	for _, m := range ms {
		b += fmt.Sprintf("%s%q%d%s;", m.Name, m.Value, m.Type, m.String())
	}
	return b
}

func filterTime(ss []model.TimeSeries, mint, maxt int64) []model.TimeSeries {
	out := make([]model.TimeSeries, 0, len(ss))
	for _, s := range ss {
		var pts []model.Sample
		for _, p := range s.Samples {
			if p.T >= mint && p.T <= maxt {
				pts = append(pts, p)
			}
		}
		if len(pts) > 0 {
			out = append(out, model.TimeSeries{Labels: s.Labels, Samples: pts})
		}
	}
	return out
}
