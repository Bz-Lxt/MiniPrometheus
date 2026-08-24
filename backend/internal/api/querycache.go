package api

import (
	"sync"
	"time"

	"github.com/alkaid/miniprometheus/internal/promql"
)

type cacheEntry struct {
	at   time.Time
	res  promql.Result
}

type queryCache struct {
	mu sync.Mutex
	m  map[string]cacheEntry
	ttl time.Duration
}

func newQueryCache(ttl time.Duration) *queryCache {
	return &queryCache{m: map[string]cacheEntry{}, ttl: ttl}
}

func (c *queryCache) get(key string) (promql.Result, bool) {
	if c == nil {
		return promql.Result{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Since(e.at) > c.ttl {
		delete(c.m, key)
		return promql.Result{}, false
	}
	return e.res, true
}

func (c *queryCache) put(key string, res promql.Result) {
	if c == nil || res.Err != nil {
		return
	}
	c.mu.Lock()
	c.m[key] = cacheEntry{at: time.Now(), res: res}
	if len(c.m) > 256 {
		var oldest string
		var t time.Time
		first := true
		for k, e := range c.m {
			if first || e.at.Before(t) {
				oldest, t, first = k, e.at, false
			}
		}
		delete(c.m, oldest)
	}
	c.mu.Unlock()
}

func cacheKey(expr string, start, end, step int64) string {
	return expr + "|" + itoa64(start) + "|" + itoa64(end) + "|" + itoa64(step)
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
