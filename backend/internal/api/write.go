package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alkaid/miniprometheus/internal/remote"
)

func (s *Server) handleWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		promErr(w, 405, "bad_data", "POST required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		promErr(w, 400, "bad_data", err.Error())
		return
	}
	var req remote.WriteRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.Unmarshal(body, &req); err != nil {
			promErr(w, 400, "bad_data", err.Error())
			return
		}
	} else {
		req, err = remote.Decode(body)
		if err != nil {
			if json.Unmarshal(body, &req) != nil {
				promErr(w, 400, "bad_data", err.Error())
				return
			}
		}
	}
	if s.Role == "gateway" && s.Shards != nil {
		sp := s.Shards.Write(req)
		s.Reg.Add("mp_samples_in", int64(sp.Sent))
		if sp.Partial {
			writeJSON(w, 202, map[string]any{"status": "partial", "sent": sp.Sent, "failed": sp.Failed})
			return
		}
		writeJSON(w, 204, map[string]any{"status": "success", "sent": sp.Sent})
		return
	}
	n := 0
	for _, ser := range req.Series {
		if err := s.Head.AppendMany(ser.Labels, ser.Samples); err != nil {
			promErr(w, 500, "internal", err.Error())
			return
		}
		n += len(ser.Samples)
	}
	s.Reg.Add("mp_samples_in", int64(n))
	w.WriteHeader(http.StatusNoContent)
}
