package api

import (
	"strconv"
	"time"
)

func parseRFCOrUnix(v string) (int64, bool) {
	if v == "" {
		return 0, false
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil {
		if n > 1e12 {
			return int64(n), true
		}
		return int64(n * 1000), true
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t.UnixMilli(), true
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UnixMilli(), true
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.FixedZone("CST", 8*3600)); err == nil {
		return t.UnixMilli(), true
	}
	return 0, false
}
