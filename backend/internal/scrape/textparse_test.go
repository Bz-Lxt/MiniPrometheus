package scrape

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseText(t *testing.T) {
	in := `
# HELP x demo
http_requests_total{job="api",status="500"} 12
node_cpu_usage{instance="a"} +Inf
broken
`
	ps, err := ParseText(strings.NewReader(in))
	require.NoError(t, err)
	require.Len(t, ps, 2)
	require.Equal(t, "http_requests_total", ps[0].Labels.Get("__name__"))
	require.Equal(t, "500", ps[0].Labels.Get("status"))
	require.Equal(t, 12.0, ps[0].Value)
}
