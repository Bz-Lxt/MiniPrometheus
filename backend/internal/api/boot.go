package api

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alkaid/miniprometheus/internal/block"
	"github.com/alkaid/miniprometheus/internal/cluster"
	"github.com/alkaid/miniprometheus/internal/config"
	"github.com/alkaid/miniprometheus/internal/head"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/metrics"
	"github.com/alkaid/miniprometheus/internal/shard"
	"github.com/alkaid/miniprometheus/internal/wal"
)

func RunStorage(cfg config.Config) error {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	walDir := filepath.Join(cfg.DataDir, "wal")
	if _, err := wal.RepairTail(walDir); err != nil {
		logger.L().Warn("wal repair", "err", err.Error())
	}
	w, err := wal.Open(walDir)
	if err != nil {
		return err
	}
	h := head.New(w)
	n, err := wal.Replay(filepath.Join(cfg.DataDir, "wal"), h)
	if err != nil {
		logger.L().Warn("wal replay incomplete", "err", err.Error(), "samples", n)
	} else {
		logger.L().Info("wal replayed", "samples", n)
	}
	bs, err := block.OpenStore(filepath.Join(cfg.DataDir, "blocks"))
	if err != nil {
		return err
	}
	srv := &Server{
		Cfg:     cfg,
		Role:    "storage",
		Head:    h,
		Blocks:  bs,
		Reg:     metrics.New(),
		Cluster: &cluster.State{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go compactLoop(ctx, cfg, h, bs)
	return serve(cfg, srv, func() {
		h.FlushAll()
		_ = w.Close()
	})
}

func RunGateway(cfg config.Config) error {
	srv := &Server{
		Cfg:     cfg,
		Role:    "gateway",
		Shards:  shard.NewClient(cfg.Shards),
		Reg:     metrics.New(),
		Cluster: &cluster.State{},
	}
	return serve(cfg, srv, nil)
}

func compactLoop(ctx context.Context, cfg config.Config, h *head.Head, bs *block.Store) {
	t := time.NewTicker(cfg.HeadBlockDur)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := block.CompactHead(h, bs); err != nil {
				logger.L().Warn("compact failed", "err", err.Error())
			}
			cut := time.Now().Add(-cfg.Retention).UnixMilli()
			bs.Expire(cut)
		}
	}
}

func serve(cfg config.Config, api *Server, onStop func()) error {
	hs := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Handler(), ReadHeaderTimeout: 8 * time.Second}
	errCh := make(chan error, 1)
	go func() {
		logger.L().Info("listen", "addr", cfg.HTTPAddr, "role", api.Role)
		errCh <- hs.ListenAndServe()
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-sig:
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = hs.Shutdown(ctx)
	if onStop != nil {
		onStop()
	}
	return nil
}
