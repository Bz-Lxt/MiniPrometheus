package metrics

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

type Registry struct {
	mu sync.Mutex
	c  map[string]*atomic.Int64
	g  map[string]*atomic.Value
}

func New() *Registry {
	return &Registry{c: make(map[string]*atomic.Int64), g: make(map[string]*atomic.Value)}
}

func (r *Registry) Add(name string, n int64) {
	r.mu.Lock()
	c, ok := r.c[name]
	if !ok {
		c = &atomic.Int64{}
		r.c[name] = c
	}
	r.mu.Unlock()
	c.Add(n)
}

func (r *Registry) Set(name string, v float64) {
	r.mu.Lock()
	g, ok := r.g[name]
	if !ok {
		g = &atomic.Value{}
		r.g[name] = g
	}
	r.mu.Unlock()
	g.Store(v)
}

func (r *Registry) Render(extra string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	b.WriteString("# TYPE mp_info gauge\nmp_info{component=\"miniprometheus\"} 1\n")
	for k, c := range r.c {
		fmt.Fprintf(&b, "# TYPE %s counter\n%s %d\n", k, k, c.Load())
	}
	for k, g := range r.g {
		v, _ := g.Load().(float64)
		fmt.Fprintf(&b, "# TYPE %s gauge\n%s %g\n", k, k, v)
	}
	if extra != "" {
		b.WriteString(extra)
	}
	return b.String()
}
