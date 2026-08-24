package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Role          string
	HTTPAddr      string
	LogLevel      string
	DataDir       string
	ShardID       int
	ShardCount    int
	Shards        []string
	WriteURL      string
	ScrapeMode    string
	ScrapeTargets []string
	ScrapeEvery   time.Duration
	SyntheticRate int
	Retention     time.Duration
	HeadBlockDur  time.Duration
	QueryTimeout  time.Duration
	MaxSamples    int
	MaxQueries    int
	CORSOrigins   []string
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func Load() Config {
	return Config{
		Role:          getenv("MP_ROLE", "storage"),
		HTTPAddr:      getenv("MP_HTTP_ADDR", ":8080"),
		LogLevel:      getenv("MP_LOG_LEVEL", "info"),
		DataDir:       getenv("MP_DATA_DIR", "/tmp/miniprometheus"),
		ShardID:       getenvInt("MP_SHARD_ID", 0),
		ShardCount:    getenvInt("MP_SHARD_COUNT", 1),
		Shards:        split(getenv("MP_SHARDS", "")),
		WriteURL:      getenv("MP_WRITE_URL", "http://127.0.0.1:8080/api/v1/write"),
		ScrapeMode:    getenv("MP_SCRAPE_MODE", "both"),
		ScrapeTargets: split(getenv("MP_SCRAPE_TARGETS", "")),
		ScrapeEvery:   time.Duration(getenvInt("MP_SCRAPE_EVERY_MS", 5000)) * time.Millisecond,
		SyntheticRate: getenvInt("MP_SYNTHETIC_RATE", 4000),
		Retention:     time.Duration(getenvInt("MP_RETENTION_HOURS", 6)) * time.Hour,
		HeadBlockDur:  time.Duration(getenvInt("MP_HEAD_BLOCK_MIN", 2)) * time.Minute,
		QueryTimeout:  time.Duration(getenvInt("MP_QUERY_TIMEOUT_MS", 8000)) * time.Millisecond,
		MaxSamples:    getenvInt("MP_MAX_SAMPLES", 2_000_000),
		MaxQueries:    getenvInt("MP_MAX_QUERIES", 16),
		CORSOrigins:   split(getenv("MP_CORS_ORIGINS", "")),
	}
}

func split(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
