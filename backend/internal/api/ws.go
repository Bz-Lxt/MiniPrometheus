package api

import (
	"net/http"
	"time"

	"github.com/alkaid/miniprometheus/internal/clock"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/gorilla/websocket"
)

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if !AllowOrigin(r, s.Cfg.CORSOrigins) {
		http.Error(w, "origin denied", http.StatusForbidden)
		return
	}
	up := websocket.Upgrader{
		CheckOrigin: func(req *http.Request) bool {
			return AllowOrigin(req, s.Cfg.CORSOrigins)
		},
	}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		logger.L().Warn("ws upgrade", "err", err.Error())
		return
	}
	defer conn.Close()
	query := r.URL.Query().Get("query")
	if query == "" {
		query = "node_cpu_usage"
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			end := clock.NowUnixMilli()
			start := end - 60_000
			res := s.exec(r.Context(), query, start, end, 5000, end)
			if res.Err != nil {
				_ = conn.WriteJSON(map[string]any{"error": res.Err.Error(), "partial": res.Partial})
				continue
			}
			if err := conn.WriteJSON(map[string]any{
				"t":       clock.Format(end),
				"partial": res.Partial,
				"data":    res.Value,
			}); err != nil {
				return
			}
		}
	}
}
