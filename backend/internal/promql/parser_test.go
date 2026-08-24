package promql

import (
	"context"
	"testing"

	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/stretchr/testify/require"
)

func TestParseSubset(t *testing.T) {
	cases := []string{
		`http_requests{status="500"}`,
		`rate(http_requests{job="api"}[5m])`,
		`sum by (job) (rate(http_requests[1m]))`,
		`avg_over_time(node_cpu_usage[2m])`,
		`topk(3, http_requests)`,
		`histogram_quantile(0.95, http_bucket)`,
		`node_cpu_usage + 1`,
	}
	for _, c := range cases {
		n, err := Parse(c)
		require.NoError(t, err, c)
		p, err := PlanOf(n)
		require.NoError(t, err, c)
		require.NotNil(t, p)
	}
}

func TestParseErrorPosition(t *testing.T) {
	_, err := Parse(`rate(foo`)
	require.Error(t, err)
	pe, ok := err.(*ParseError)
	require.True(t, ok)
	require.Greater(t, pe.Col, 0)
}

func TestEngineRate(t *testing.T) {
	eng := &Engine{Src: func(ms []*index.Matcher, mint, maxt int64) ([]model.TimeSeries, index.LookupStat) {
		return []model.TimeSeries{{
			Labels: model.FromMap("http_requests", map[string]string{"job": "api"}),
			Samples: []model.Sample{
				{T: 1000, V: 10},
				{T: 2000, V: 20},
				{T: 3000, V: 40},
			},
		}}, index.LookupStat{HitIndex: true, Series: 1}
	}}
	res := eng.Exec(context.Background(), `rate(http_requests[5m])`, 0, 3000, 0, 3000)
	require.NoError(t, res.Err)
	require.Equal(t, ValVector, res.Value.Kind)
	require.Len(t, res.Value.Series, 1)
	require.InDelta(t, 15.0, res.Value.Series[0].Points[0].V, 0.01)
	require.Equal(t, "rate", res.Profile.Op)
}
