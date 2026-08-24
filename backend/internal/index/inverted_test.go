package index

import (
	"testing"

	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/stretchr/testify/require"
)

func TestLookupMatchers(t *testing.T) {
	iv := NewInverted()
	id1, _ := iv.GetOrCreate(model.FromMap("http_requests", map[string]string{"instance": "api-01", "status": "500"}))
	id2, _ := iv.GetOrCreate(model.FromMap("http_requests", map[string]string{"instance": "api-02", "status": "200"}))
	_, _ = id1, id2
	ms := []*Matcher{{Name: "status", Type: MatchEqual, Value: "500"}}
	bm, st := iv.Lookup(ms)
	require.True(t, st.HitIndex)
	require.Equal(t, 1, bm.Cardinality())
	require.True(t, bm.Contains(uint32(id1)))

	ms = []*Matcher{{Name: "status", Type: MatchNotEqual, Value: "500"}}
	bm, _ = iv.Lookup(ms)
	require.True(t, bm.Contains(uint32(id2)))
	require.False(t, bm.Contains(uint32(id1)))
}
