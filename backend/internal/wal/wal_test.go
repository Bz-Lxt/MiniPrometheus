package wal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/stretchr/testify/require"
)

type sink struct {
	series  int
	samples int
}

func (s *sink) RestoreSeries(id uint32, ls model.Labels) { s.series++ }
func (s *sink) RestoreSample(id uint32, t int64, v float64) {
	s.samples++
}

func TestWALRoundTripAndDoubleClose(t *testing.T) {
	logger.Init("error", nil)
	dir := t.TempDir()
	w, err := Open(dir)
	require.NoError(t, err)
	require.NoError(t, w.LogSeries(1, model.FromMap("m", map[string]string{"a": "b"})))
	require.NoError(t, w.LogSample(1, 100, 1.5))
	require.NoError(t, w.LogSample(1, 200, 2.5))
	require.NoError(t, w.Close())
	require.NoError(t, w.Close())
	s := &sink{}
	n, err := Replay(dir, s)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, 1, s.series)
	_ = filepath.Join(dir, "x")
	_ = os.Remove
}
