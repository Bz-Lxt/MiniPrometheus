package api

import (
	"net/http"
	"strings"

	"github.com/alkaid/miniprometheus/internal/promql"
)

func (s *Server) handleSuggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	out := map[string]any{"functions": promql.Catalog(), "metrics": []string{}}
	if s.Head != nil {
		names := s.Head.Index().LabelValues("__name__")
		if q != "" {
			filt := names[:0]
			for _, n := range names {
				if strings.Contains(n, q) {
					filt = append(filt, n)
				}
			}
			names = append([]string(nil), filt...)
		}
		out["metrics"] = names
		out["labels"] = s.Head.Index().LabelNames()
	}
	writeJSON(w, 200, out)
}
