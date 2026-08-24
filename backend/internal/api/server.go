package api

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alkaid/miniprometheus/internal/block"
	"github.com/alkaid/miniprometheus/internal/cluster"
	"github.com/alkaid/miniprometheus/internal/config"
	"github.com/alkaid/miniprometheus/internal/head"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/metrics"
	"github.com/alkaid/miniprometheus/internal/shard"
)

type Server struct {
	Cfg     config.Config
	Role    string
	Head    *head.Head
	Blocks  *block.Store
	Shards  *shard.Client
	Reg     *metrics.Registry
	Cluster *cluster.State

	inflight atomic.Int64
	mu       sync.Mutex
	degraded []string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/query", s.handleQuery)
	mux.HandleFunc("/api/v1/query_range", s.handleQueryRange)
	mux.HandleFunc("/api/v1/query_profile", s.handleQuery)
	mux.HandleFunc("/api/v1/series", s.handleSeries)
	mux.HandleFunc("/api/v1/labels", s.handleLabels)
	mux.HandleFunc("/api/v1/label/", s.handleLabelValues)
	mux.HandleFunc("/api/v1/write", s.handleWrite)
	mux.HandleFunc("/api/v1/select", s.handleSelect)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/cluster", s.handleCluster)
	mux.HandleFunc("/api/v1/index/explain", s.handleIndexExplain)
	mux.HandleFunc("/api/v1/suggest", s.handleSuggest)
	mux.HandleFunc("/ws/stream", s.handleWS)
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			if !AllowOrigin(r, s.Cfg.CORSOrigins) {
				http.Error(w, "origin denied", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && AllowOrigin(r, s.Cfg.CORSOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		ww := &wrap{ResponseWriter: w, status: 200}
		start := time.Now()
		next.ServeHTTP(ww, r)
		logger.L().Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) acquire() bool {
	n := s.inflight.Add(1)
	if s.Cfg.MaxQueries > 0 && int(n) > s.Cfg.MaxQueries {
		s.inflight.Add(-1)
		return false
	}
	return true
}

func (s *Server) release() { s.inflight.Add(-1) }

func (s *Server) setPartial(failed []string) {
	s.mu.Lock()
	s.degraded = failed
	s.mu.Unlock()
}

func (s *Server) lastPartial() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.degraded) > 0
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "role": s.Role, "time": cluster.Now()})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	extra := ""
	if s.Head != nil {
		st := s.Head.Stats()
		s.Reg.Set("mp_head_series", float64(st.Series))
		s.Reg.Set("mp_head_samples", float64(st.Samples))
		s.Reg.Set("mp_bytes_per_sample", st.BPS)
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(s.Reg.Render(extra)))
}
