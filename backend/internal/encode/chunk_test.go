package encode

import (
	"math"
	"testing"

	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/stretchr/testify/require"
)

func TestRoundTripSpecials(t *testing.T) {
	ss := []model.Sample{
		{T: 1_700_000_000_000, V: 1.0},
		{T: 1_700_000_001_000, V: 1.0},
		{T: 1_700_000_002_000, V: 1.25},
		{T: 1_700_000_003_000, V: math.NaN()},
		{T: 1_700_000_004_000, V: math.Inf(1)},
		{T: 1_700_000_005_000, V: math.Inf(-1)},
		{T: 1_700_000_006_000, V: math.Copysign(0, -1)},
		{T: 1_700_000_007_000, V: math.SmallestNonzeroFloat64},
	}
	b, st, err := Encode(ss)
	require.NoError(t, err)
	require.Greater(t, st.Points, 0)
	got, err := Decode(b)
	require.NoError(t, err)
	require.Equal(t, len(ss), len(got))
	for i := range ss {
		require.Equal(t, ss[i].T, got[i].T)
		require.Equal(t, math.Float64bits(ss[i].V), math.Float64bits(got[i].V), i)
	}
}

func TestSmoothCompression(t *testing.T) {
	ss := make([]model.Sample, 120)
	t0 := int64(1_700_000_000_000)
	base := math.Float64bits(62.5)
	for i := range ss {
		// 1s 规则采样 + 极缓变（ULP 级），贴近真实 CPU/内存曲线的位级局部性
		ss[i] = model.Sample{T: t0 + int64(i)*1000, V: math.Float64frombits(base + uint64(i/8))}
	}
	b, st, err := Encode(ss)
	require.NoError(t, err)
	require.LessOrEqual(t, st.BytesPerSample(), 2.0, "smooth series must compress ≤ 2.0 B/sample, got %f (%d bytes)", st.BytesPerSample(), len(b))
	got, err := Decode(b)
	require.NoError(t, err)
	require.Equal(t, ss, got)
}
