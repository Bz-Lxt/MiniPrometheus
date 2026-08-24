package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alkaid/miniprometheus/internal/config"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/remote"
	"github.com/alkaid/miniprometheus/internal/scrape"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sink := func(ss []remote.TimeSeries) {
		if err := remote.Push(cfg.WriteURL, remote.WriteRequest{Series: ss}); err != nil {
			logger.L().Warn("remote write", "err", err.Error(), "n", len(ss))
		}
	}

	switch cfg.ScrapeMode {
	case "real":
		go runScrape(ctx, cfg, sink)
	case "synthetic":
		go runSynth(ctx, cfg, sink)
	default:
		go runScrape(ctx, cfg, sink)
		go runSynth(ctx, cfg, sink)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","role":"agent"}`))
	})
	hs := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.L().Info("agent listen", "addr", cfg.HTTPAddr, "mode", cfg.ScrapeMode)
		_ = hs.ListenAndServe()
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	_ = hs.Shutdown(context.Background())
}

func runScrape(ctx context.Context, cfg config.Config, sink func([]remote.TimeSeries)) {
	sc := &scrape.Scraper{Targets: cfg.ScrapeTargets, Job: "minip-agent", Every: cfg.ScrapeEvery, Sink: sink}
	sc.Run(ctx)
}

func runSynth(ctx context.Context, cfg config.Config, sink func([]remote.TimeSeries)) {
	g := &scrape.Synthetic{Rate: cfg.SyntheticRate, Sink: sink}
	g.Run(ctx)
}
