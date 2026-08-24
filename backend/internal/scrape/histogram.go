package scrape

import (
	"strconv"
	"strings"

	"github.com/alkaid/miniprometheus/internal/model"
)

func IsHistogramBucket(ls model.Labels) bool {
	n := ls.Get(model.MetricName)
	return strings.HasSuffix(n, "_bucket") && ls.Get("le") != ""
}

func IsHistogramSum(ls model.Labels) bool {
	return strings.HasSuffix(ls.Get(model.MetricName), "_sum")
}

func IsHistogramCount(ls model.Labels) bool {
	return strings.HasSuffix(ls.Get(model.MetricName), "_count")
}

func ParseLE(ls model.Labels) (float64, bool) {
	le := ls.Get("le")
	if le == "" {
		return 0, false
	}
	if le == "+Inf" {
		return 0, true
	}
	v, err := strconv.ParseFloat(le, 64)
	return v, err == nil
}

func FamilyName(ls model.Labels) string {
	n := ls.Get(model.MetricName)
	for _, suf := range []string{"_bucket", "_sum", "_count"} {
		if strings.HasSuffix(n, suf) {
			return strings.TrimSuffix(n, suf)
		}
	}
	return n
}
