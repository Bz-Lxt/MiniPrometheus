package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alkaid/miniprometheus/internal/head"
	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/promql"
)

func (s *Server) handleSeries(w http.ResponseWriter, r *http.Request) {
	matches := r.URL.Query()["match[]"]
	var all []model.Labels
	if s.Role == "gateway" && s.Shards != nil {
		bodies, failed := s.Shards.GetJSON(r.Context(), "/api/v1/series", r.URL.Query())
		s.setPartial(failed)
		for _, b := range bodies {
			var wrap struct {
				Data []map[string]string `json:"data"`
			}
			if json.Unmarshal(b, &wrap) == nil {
				for _, m := range wrap.Data {
					all = append(all, model.FromMap(m[model.MetricName], m))
				}
			}
		}
	} else {
		if len(matches) == 0 {
			all = s.Head.SeriesMeta(nil)
		} else {
			for _, m := range matches {
				ms, err := parseSelectorMatchers(m)
				if err != nil {
					promErr(w, 400, "bad_data", err.Error())
					return
				}
				all = append(all, s.Head.SeriesMeta(ms)...)
			}
		}
	}
	seen := map[string]struct{}{}
	out := make([]map[string]string, 0, len(all))
	for _, ls := range all {
		k := ls.String()
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, ls.Map())
	}
	promOK(w, out, nil)
}

func (s *Server) handleLabels(w http.ResponseWriter, r *http.Request) {
	if s.Role == "gateway" && s.Shards != nil {
		bodies, failed := s.Shards.GetJSON(r.Context(), "/api/v1/labels", nil)
		s.setPartial(failed)
		var parts [][]string
		for _, b := range bodies {
			var wrap struct {
				Data []string `json:"data"`
			}
			if json.Unmarshal(b, &wrap) == nil {
				parts = append(parts, wrap.Data)
			}
		}
		promOK(w, mergeStringSets(parts), nil)
		return
	}
	promOK(w, s.Head.Index().LabelNames(), nil)
}

func (s *Server) handleLabelValues(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/label/")
	name = strings.TrimSuffix(name, "/values")
	if name == "" {
		promErr(w, 400, "bad_data", "missing label name")
		return
	}
	if s.Role == "gateway" && s.Shards != nil {
		bodies, failed := s.Shards.GetJSON(r.Context(), "/api/v1/label/"+name+"/values", nil)
		s.setPartial(failed)
		var parts [][]string
		for _, b := range bodies {
			var wrap struct {
				Data []string `json:"data"`
			}
			if json.Unmarshal(b, &wrap) == nil {
				parts = append(parts, wrap.Data)
			}
		}
		promOK(w, mergeStringSets(parts), nil)
		return
	}
	promOK(w, s.Head.Index().LabelValues(name), nil)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	st := map[string]any{"role": s.Role, "time": time.Now().Format("2006-01-02 15:04:05")}
	if s.Head != nil {
		st["head"] = s.Head.Stats()
	}
	if s.Blocks != nil {
		st["blocks"] = s.Blocks.List()
	}
	if s.Role == "gateway" && s.Shards != nil {
		bodies, failed := s.Shards.GetJSON(r.Context(), "/api/v1/status", nil)
		s.setPartial(failed)
		var series, samples, bin, bout int64
		for _, b := range bodies {
			var wrap struct {
				Head head.Stats `json:"head"`
			}
			if json.Unmarshal(b, &wrap) == nil {
				series += int64(wrap.Head.Series)
				samples += wrap.Head.Samples
				bin += wrap.Head.BytesIn
				bout += wrap.Head.BytesOut
			}
		}
		bps := 0.0
		if samples > 0 && bout > 0 {
			bps = float64(bout) / float64(samples)
		}
		st["head"] = map[string]any{"series": series, "samples": samples, "bytes_in": bin, "bytes_out": bout, "bytes_per_sample": bps}
		st["shard_status_raw"] = len(bodies)
	}
	s.mu.Lock()
	st["degraded"] = append([]string(nil), s.degraded...)
	s.mu.Unlock()
	writeJSON(w, 200, st)
}

func (s *Server) handleCluster(w http.ResponseWriter, r *http.Request) {
	if s.Shards != nil {
		writeJSON(w, 200, map[string]any{"shards": s.Shards.Health(r.Context()), "degraded": s.degraded})
		return
	}
	writeJSON(w, 200, map[string]any{"shards": []any{}, "role": s.Role})
}

func (s *Server) handleIndexExplain(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("query")
	if q == "" {
		q = r.URL.Query().Get("match[]")
	}
	if s.Role == "gateway" && s.Shards != nil {
		bodies, failed := s.Shards.GetJSON(r.Context(), "/api/v1/index/explain", r.URL.Query())
		s.setPartial(failed)
		var series int
		var us int64
		for _, b := range bodies {
			var wrap struct {
				Series     int   `json:"series"`
				DurationUS int64 `json:"duration_us"`
			}
			if json.Unmarshal(b, &wrap) == nil {
				series += wrap.Series
				us += wrap.DurationUS
			}
		}
		writeJSON(w, 200, map[string]any{"query": q, "series": series, "duration_us": us, "partial": len(failed) > 0, "shards": len(bodies)})
		return
	}
	ms, err := parseSelectorMatchers(q)
	if err != nil {
		promErr(w, 400, "bad_data", err.Error())
		return
	}
	if s.Head == nil {
		promErr(w, 400, "bad_data", "no local index")
		return
	}
	start := time.Now()
	bm, st := s.Head.Index().Lookup(ms)
	writeJSON(w, 200, map[string]any{
		"matchers":     matcherStr(ms),
		"series":       bm.Cardinality(),
		"duration_us":  time.Since(start).Microseconds(),
		"lookup":       st,
		"sample_ids":   take(bm.ToArray(), 32),
	})
}

func parseSelectorMatchers(q string) ([]*index.Matcher, error) {
	if q == "" {
		return nil, nil
	}
	n, err := promql.Parse(q)
	if err != nil {
		return nil, err
	}
	sel, ok := n.(*promql.Selector)
	if !ok {
		p, err := promql.PlanOf(n)
		if err != nil {
			return nil, err
		}
		return p.Matchers, nil
	}
	return sel.Matchers, nil
}

func matcherStr(ms []*index.Matcher) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.String()
	}
	return out
}

func take(ids []uint32, n int) []uint32 {
	if len(ids) <= n {
		return ids
	}
	return ids[:n]
}
