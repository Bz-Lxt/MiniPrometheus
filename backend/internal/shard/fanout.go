package shard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/alkaid/miniprometheus/internal/remote"
)

type Client struct {
	Endpoints []string
	HTTP      *http.Client
}

func NewClient(eps []string) *Client {
	return &Client{Endpoints: eps, HTTP: &http.Client{Timeout: 6 * time.Second}}
}

type WriteSplit struct {
	Sent    int
	Failed  []string
	Partial bool
}

func (c *Client) Write(req remote.WriteRequest) WriteSplit {
	n := len(c.Endpoints)
	if n == 0 {
		return WriteSplit{Failed: []string{"no shards"}}
	}
	buckets := make([]remote.WriteRequest, n)
	for _, s := range req.Series {
		i := Index(s.Labels, n)
		buckets[i].Series = append(buckets[i].Series, s)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string
	sent := 0
	for i, ep := range c.Endpoints {
		if len(buckets[i].Series) == 0 {
			continue
		}
		wg.Add(1)
		go func(ep string, body remote.WriteRequest) {
			defer wg.Done()
			if err := remote.Push(ep+"/api/v1/write", body); err != nil {
				mu.Lock()
				failed = append(failed, ep+": "+err.Error())
				mu.Unlock()
				return
			}
			mu.Lock()
			sent += len(body.Series)
			mu.Unlock()
		}(ep, buckets[i])
	}
	wg.Wait()
	return WriteSplit{Sent: sent, Failed: failed, Partial: len(failed) > 0}
}

type ShardQuery struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error"`
	Extra  map[string]any  `json:"-"`
}

func (c *Client) GetJSON(ctx context.Context, path string, q url.Values) ([]json.RawMessage, []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var bodies []json.RawMessage
	var failed []string
	for _, ep := range c.Endpoints {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()
			u := ep + path
			if len(q) > 0 {
				u += "?" + q.Encode()
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			if err != nil {
				mu.Lock()
				failed = append(failed, ep)
				mu.Unlock()
				return
			}
			resp, err := c.HTTP.Do(req)
			if err != nil {
				mu.Lock()
				failed = append(failed, ep+": "+err.Error())
				mu.Unlock()
				return
			}
			defer resp.Body.Close()
			b, err := io.ReadAll(resp.Body)
			if err != nil || resp.StatusCode/100 != 2 {
				mu.Lock()
				failed = append(failed, fmt.Sprintf("%s status %d", ep, resp.StatusCode))
				mu.Unlock()
				return
			}
			mu.Lock()
			bodies = append(bodies, b)
			mu.Unlock()
		}(ep)
	}
	wg.Wait()
	return bodies, failed
}

func (c *Client) Post(ctx context.Context, path string, body []byte) error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("no shards")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoints[0]+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Health(ctx context.Context) []map[string]any {
	out := make([]map[string]any, 0, len(c.Endpoints))
	for i, ep := range c.Endpoints {
		item := map[string]any{"id": i, "endpoint": ep, "healthy": false}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep+"/health", nil)
		if err != nil {
			out = append(out, item)
			continue
		}
		resp, err := c.HTTP.Do(req)
		if err == nil && resp.StatusCode == 200 {
			item["healthy"] = true
			_ = resp.Body.Close()
		}
		out = append(out, item)
	}
	return out
}
