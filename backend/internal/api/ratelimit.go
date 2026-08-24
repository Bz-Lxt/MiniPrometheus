package api

import (
	"sync"
	"time"
)

type window struct {
	t []time.Time
}

type limiter struct {
	mu   sync.Mutex
	byIP map[string]*window
	max  int
	span time.Duration
}

func newLimiter(max int, span time.Duration) *limiter {
	return &limiter{byIP: map[string]*window{}, max: max, span: span}
}

func (l *limiter) allow(ip string) bool {
	if l == nil || l.max <= 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.byIP[ip]
	if w == nil {
		w = &window{}
		l.byIP[ip] = w
	}
	cut := now.Add(-l.span)
	kept := w.t[:0]
	for _, t := range w.t {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	w.t = kept
	if len(w.t) >= l.max {
		return false
	}
	w.t = append(w.t, now)
	return true
}
