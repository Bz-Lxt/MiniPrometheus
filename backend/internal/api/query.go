package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/alkaid/miniprometheus/internal/downsample"
	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/promql"
)

func (s *Server) selectLocal(ms []*index.Matcher, mint, maxt int64) ([]model.TimeSeries, index.LookupStat) {
	hs, st := s.Head.Query(ms, mint, maxt)
	var bs []model.TimeSeries
	if s.Blocks != nil {
		bs = s.Blocks.Query(ms, mint, maxt)
	}
	return mergeSeries([][]model.TimeSeries{hs, bs}), st
}

func (s *Server) engine() *promql.Engine {
	return &promql.Engine{
		Src:     s.selectLocal,
		Timeout: s.Cfg.QueryTimeout,
		MaxScan: s.Cfg.MaxSamples,
	}
}

func parseTimeParam(v string, def int64) int64 {
	if v == "" {
		return def
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil {
		if n > 1e12 {
			return int64(n)
		}
		return int64(n * 1000)
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UnixMilli()
	}
	return def
}

func parseStep(v string, def int64) int64 {
	if v == "" {
		return def
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil {
		if n < 100 {
			return int64(n * 1000)
		}
		return int64(n)
	}
	if d, err := promql.ParseDuration(v); err == nil {
		return d
	}
	return def
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if !s.acquire() {
		promErr(w, http.StatusTooManyRequests, "too_many", "max concurrent queries")
		return
	}
	defer s.release()
	q := r.URL.Query().Get("query")
	if q == "" {
		promErr(w, 400, "bad_data", "missing query")
		return
	}
	t := parseTimeParam(r.URL.Query().Get("time"), time.Now().UnixMilli())
	res := s.exec(r.Context(), q, 0, t, 0, t)
	s.respondQuery(w, res, false)
}

func (s *Server) handleQueryRange(w http.ResponseWriter, r *http.Request) {
	if !s.acquire() {
		promErr(w, http.StatusTooManyRequests, "too_many", "max concurrent queries")
		return
	}
	defer s.release()
	q := r.URL.Query().Get("query")
	if q == "" {
		promErr(w, 400, "bad_data", "missing query")
		return
	}
	end := parseTimeParam(r.URL.Query().Get("end"), time.Now().UnixMilli())
	start := parseTimeParam(r.URL.Query().Get("start"), end-15*60*1000)
	step := parseStep(r.URL.Query().Get("step"), 15_000)
	maxPts, _ := strconv.Atoi(r.URL.Query().Get("max_points"))
	res := s.exec(r.Context(), q, start, end, step, end)
	if maxPts > 0 && res.Err == nil {
		for i := range res.Value.Series {
			ss := make([]model.Sample, len(res.Value.Series[i].Points))
			for j, p := range res.Value.Series[i].Points {
				ss[j] = model.Sample{T: p.T, V: p.V}
			}
			ss = downsample.Limit(ss, maxPts)
			pts := make([]promql.SamplePoint, len(ss))
			for j, p := range ss {
				pts[j] = promql.SamplePoint{T: p.T, V: p.V}
			}
			res.Value.Series[i].Points = pts
		}
	}
	s.respondQuery(w, res, true)
}

func (s *Server) exec(ctx context.Context, expr string, start, end, step, eval int64) promql.Result {
	if s.Role == "gateway" && s.Shards != nil {
		src := func(ms []*index.Matcher, mint, maxt int64) ([]model.TimeSeries, index.LookupStat) {
			return s.fanSelect(ctx, ms, mint, maxt)
		}
		eng := &promql.Engine{Src: src, Timeout: s.Cfg.QueryTimeout, MaxScan: s.Cfg.MaxSamples}
		res := eng.Exec(ctx, expr, start, end, step, eval)
		res.Partial = s.lastPartial()
		return res
	}
	return s.engine().Exec(ctx, expr, start, end, step, eval)
}

func (s *Server) fanSelect(ctx context.Context, ms []*index.Matcher, mint, maxt int64) ([]model.TimeSeries, index.LookupStat) {
	raw, _ := json.Marshal(ms)
	q := map[string][]string{
		"matchers": {string(raw)},
		"start":    {strconv.FormatInt(mint, 10)},
		"end":      {strconv.FormatInt(maxt, 10)},
	}
	bodies, failed := s.Shards.GetJSON(ctx, "/api/v1/select", q)
	s.setPartial(failed)
	var parts [][]model.TimeSeries
	st := index.LookupStat{HitIndex: true}
	for _, b := range bodies {
		var wrap struct {
			Status string `json:"status"`
			Data   struct {
				Result []model.TimeSeries `json:"result"`
				Stat   index.LookupStat   `json:"stat"`
			} `json:"data"`
		}
		if err := json.Unmarshal(b, &wrap); err == nil {
			parts = append(parts, wrap.Data.Result)
			st.Series += wrap.Data.Stat.Series
			st.DurationUS += wrap.Data.Stat.DurationUS
		}
	}
	return mergeSeries(parts), st
}

func (s *Server) handleSelect(w http.ResponseWriter, r *http.Request) {
	var ms []*index.Matcher
	if raw := r.URL.Query().Get("matchers"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &ms); err != nil {
			promErr(w, 400, "bad_data", err.Error())
			return
		}
	}
	start := parseTimeParam(r.URL.Query().Get("start"), 0)
	end := parseTimeParam(r.URL.Query().Get("end"), time.Now().UnixMilli())
	ts, st := s.selectLocal(ms, start, end)
	promOK(w, map[string]any{"result": ts, "stat": st}, nil)
}

func (s *Server) respondQuery(w http.ResponseWriter, res promql.Result, matrix bool) {
	if res.Err != nil {
		status := 400
		typ := "bad_data"
		if pe, ok := res.Err.(*promql.ParseError); ok {
			promErr(w, status, typ, pe.Error())
			return
		}
		if res.Err.Error() == "query timeout" || contains(res.Err.Error(), "timeout") {
			promErr(w, http.StatusGatewayTimeout, "timeout", res.Err.Error())
			return
		}
		if contains(res.Err.Error(), "max samples") {
			promErr(w, http.StatusTooManyRequests, "too_many", res.Err.Error())
			return
		}
		promErr(w, status, typ, res.Err.Error())
		return
	}
	rt := "vector"
	if matrix || res.Value.Kind == promql.ValMatrix {
		rt = "matrix"
	}
	data := promql.PromQLResult(res.Value, rt)
	extra := map[string]any{
		"profile": res.Profile,
		"partial": res.Partial,
	}
	if len(s.degraded) > 0 {
		extra["degraded_shards"] = s.degraded
	}
	promOK(w, data, extra)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
