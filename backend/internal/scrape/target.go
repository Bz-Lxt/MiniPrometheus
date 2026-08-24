package scrape

import (
	"net/url"
	"strings"
)

func NormalizeTarget(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errEmptyTarget
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/metrics"
	}
	return u.String(), nil
}

func NormalizeAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		u, err := NormalizeTarget(s)
		if err != nil {
			continue
		}
		out = append(out, u)
	}
	return out
}

type emptyTarget string

func (e emptyTarget) Error() string { return string(e) }

const errEmptyTarget emptyTarget = "empty scrape target"
