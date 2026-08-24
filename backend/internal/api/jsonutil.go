package api

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func promOK(w http.ResponseWriter, data any, extra map[string]any) {
	m := map[string]any{"status": "success", "data": data}
	for k, v := range extra {
		m[k] = v
	}
	writeJSON(w, http.StatusOK, m)
}

func promErr(w http.ResponseWriter, status int, typ, msg string) {
	writeJSON(w, status, map[string]any{
		"status":    "error",
		"errorType": typ,
		"error":     msg,
	})
}
