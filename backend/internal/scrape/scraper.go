package scrape

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alkaid/miniprometheus/internal/clock"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/remote"
)

type Scraper struct {
	Targets []string
	Job     string
	Every   time.Duration
	Sink    func([]remote.TimeSeries)
	mu      sync.Mutex
	last    int
	ok      int
	fail    int
}

func (s *Scraper) Run(ctx context.Context) {
	if s.Every <= 0 {
		s.Every = 5 * time.Second
	}
	t := time.NewTicker(s.Every)
	defer t.Stop()
	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scraper) tick(ctx context.Context) {
	for _, u := range s.Targets {
		if err := s.scrapeOne(ctx, u); err != nil {
			s.mu.Lock()
			s.fail++
			s.mu.Unlock()
			logger.L().Warn("scrape failed", "target", u, "err", err.Error())
			continue
		}
		s.mu.Lock()
		s.ok++
		s.mu.Unlock()
	}
}

func (s *Scraper) scrapeOne(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	parsed, err := ParseText(resp.Body)
	if err != nil {
		return err
	}
	now := clock.NowUnixMilli()
	out := make([]remote.TimeSeries, 0, len(parsed))
	for _, p := range parsed {
		ls := append(model.Labels(nil), p.Labels...)
		ls = append(ls, model.Label{Name: "job", Value: s.job()})
		ls = append(ls, model.Label{Name: "instance", Value: url})
		out = append(out, remote.TimeSeries{
			Labels:  model.Normalize(ls),
			Samples: []model.Sample{{T: now, V: p.Value}},
		})
	}
	s.mu.Lock()
	s.last = len(out)
	s.mu.Unlock()
	if s.Sink != nil {
		s.Sink(out)
	}
	return nil
}

func (s *Scraper) job() string {
	if s.Job == "" {
		return "minip"
	}
	return s.Job
}

func (s *Scraper) Stats() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]int{"ok": s.ok, "fail": s.fail, "last_series": s.last}
}
