package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func AllowOrigin(r *http.Request, extra []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := r.Host
	if strings.EqualFold(u.Host, host) {
		return true
	}
	for _, w := range extra {
		if strings.EqualFold(strings.TrimSpace(w), origin) || strings.EqualFold(strings.TrimSpace(w), u.Host) {
			return true
		}
	}
	h, _, _ := net.SplitHostPort(u.Host)
	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	return false
}
