package cluster

import (
	"sync"
	"time"

	"github.com/alkaid/miniprometheus/internal/clock"
)

type Node struct {
	ID       int    `json:"id"`
	Role     string `json:"role"`
	Endpoint string `json:"endpoint"`
	Healthy  bool   `json:"healthy"`
	Samples  int64  `json:"samples"`
	Series   int    `json:"series"`
	BPS      float64 `json:"bytes_per_sample"`
	Updated  string `json:"updated_at"`
}

type State struct {
	mu    sync.RWMutex
	nodes []Node
}

func (s *State) Set(nodes []Node) {
	now := clock.Now().Format("2006-01-02 15:04:05")
	for i := range nodes {
		if nodes[i].Updated == "" {
			nodes[i].Updated = now
		}
	}
	s.mu.Lock()
	s.nodes = nodes
	s.mu.Unlock()
}

func (s *State) List() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Node, len(s.nodes))
	copy(out, s.nodes)
	return out
}

func Now() string { return time.Now().In(clock.Beijing()).Format("2006-01-02 15:04:05") }
