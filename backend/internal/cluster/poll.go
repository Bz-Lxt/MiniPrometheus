package cluster

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func PollHealth(ctx context.Context, endpoints []string, every time.Duration, dst *State) {
	if every <= 0 {
		every = 5 * time.Second
	}
	t := time.NewTicker(every)
	defer t.Stop()
	cli := &http.Client{Timeout: 2 * time.Second}
	refresh := func() {
		nodes := make([]Node, 0, len(endpoints))
		for i, ep := range endpoints {
			n := Node{ID: i, Role: "storage", Endpoint: ep}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep+"/health", nil)
			if err == nil {
				if resp, err := cli.Do(req); err == nil {
					n.Healthy = resp.StatusCode == 200
					var body map[string]any
					_ = json.NewDecoder(resp.Body).Decode(&body)
					_ = resp.Body.Close()
				}
			}
			nodes = append(nodes, n)
		}
		dst.Set(nodes)
	}
	refresh()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			refresh()
		}
	}
}
