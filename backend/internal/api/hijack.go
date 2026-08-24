package api

import (
	"bufio"
	"net"
	"net/http"
)

type wrap struct {
	http.ResponseWriter
	status int
}

func (w *wrap) WriteHeader(c int) {
	w.status = c
	w.ResponseWriter.WriteHeader(c)
}

func (w *wrap) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *wrap) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errNoHijack
	}
	return h.Hijack()
}

type hijackErr string

func (e hijackErr) Error() string { return string(e) }

const errNoHijack hijackErr = "response does not implement http.Hijacker"
