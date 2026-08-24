package scrape

import (
	"context"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/alkaid/miniprometheus/internal/clock"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/remote"
)

type Synthetic struct {
	Rate    int
	Sink    func([]remote.TimeSeries)
	running atomic.Bool
	emitted atomic.Int64
}

func (g *Synthetic) Run(ctx context.Context) {
	if g.Rate <= 0 {
		g.Rate = 2000
	}
	g.running.Store(true)
	defer g.running.Store(false)
	instances := []string{"api-01", "api-02", "api-03", "worker-01", "worker-02"}
	jobs := []string{"api", "worker"}
	statuses := []string{"200", "201", "400", "500"}
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			n := g.Rate / 5
			if n < 20 {
				n = 20
			}
			now := clock.NowUnixMilli()
			batch := make([]remote.TimeSeries, 0, n)
			for i := 0; i < n; i++ {
				inst := instances[i%len(instances)]
				job := jobs[i%len(jobs)]
				st := statuses[i%len(statuses)]
				cpu := 20 + 15*math.Sin(float64(now)/7000+float64(i)) + rand.Float64()*2
				mem := 1024 + 80*math.Sin(float64(now)/11000+float64(i)*0.3)
				lat := 12 + 6*math.Sin(float64(now)/4000+float64(i)*0.7) + rand.Float64()
				req := float64(100+i%40) + float64((now/1000)%20)
				batch = append(batch,
					ts("node_cpu_usage", map[string]string{"job": job, "instance": inst, "mode": "user"}, now, cpu),
					ts("node_memory_used_bytes", map[string]string{"job": job, "instance": inst}, now, mem*1024*1024),
					ts("http_request_duration_ms", map[string]string{"job": job, "instance": inst, "status": st}, now, lat),
					ts("http_requests_total", map[string]string{"job": job, "instance": inst, "status": st}, now, req),
				)
			}
			if g.Sink != nil {
				g.Sink(batch)
			}
			g.emitted.Add(int64(len(batch)))
		}
	}
}

func ts(name string, labs map[string]string, t int64, v float64) remote.TimeSeries {
	return remote.TimeSeries{Labels: model.FromMap(name, labs), Samples: []model.Sample{{T: t, V: v}}}
}

func (g *Synthetic) Emitted() int64 { return g.emitted.Load() }
