package main

import (
	"flag"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/alkaid/miniprometheus/internal/clock"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/remote"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:8080/api/v1/write", "write url")
	rate := flag.Int("rate", 200000, "samples per second")
	secs := flag.Int("duration", 15, "seconds")
	batch := flag.Int("batch", 1000, "samples per request")
	flag.Parse()
	logger.Init("info", nil)
	var sent atomic.Int64
	deadline := time.Now().Add(time.Duration(*secs) * time.Second)
	workers := 8
	perWorker := *rate / workers
	if perWorker < *batch {
		perWorker = *batch
	}
	errCh := make(chan error, workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			interval := time.Second * time.Duration(*batch) / time.Duration(perWorker)
			for time.Now().Before(deadline) {
				loopStart := time.Now()
				now := clock.NowUnixMilli()
				n := *batch
				ss := make([]remote.TimeSeries, 0, n)
				for i := 0; i < n; i++ {
					ls := model.FromMap("loadgen_metric", map[string]string{
						"job": "loadgen", "instance": fmt.Sprintf("lg-%d", id), "i": fmt.Sprintf("%d", i%64),
					})
					v := 40 + float64((now/1000+int64(i))%17)
					ss = append(ss, remote.TimeSeries{Labels: ls, Samples: []model.Sample{{T: now, V: v}}})
				}
				if err := remote.Push(*url, remote.WriteRequest{Series: ss}); err != nil {
					errCh <- err
					return
				}
				sent.Add(int64(n))
				if d := interval - time.Since(loopStart); d > 0 {
					time.Sleep(d)
				}
			}
			errCh <- nil
		}(w)
	}
	var first error
	for i := 0; i < workers; i++ {
		if e := <-errCh; e != nil && first == nil {
			first = e
		}
	}
	fmt.Printf("sent=%d duration=%ds rate=%.0f/s err=%v\n", sent.Load(), *secs, float64(sent.Load())/float64(*secs), first)
	if first != nil {
		os.Exit(1)
	}
}
