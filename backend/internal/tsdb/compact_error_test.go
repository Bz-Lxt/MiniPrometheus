package tsdb_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alkaid/miniprometheus/internal/block"
	"github.com/alkaid/miniprometheus/internal/head"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/tsdb"
	"github.com/stretchr/testify/require"
)

func TestCompactFailureKeepsHeadDataForRetry(t *testing.T) {
	blocksDir := filepath.Join(t.TempDir(), "blocks")
	store, err := block.OpenStore(blocksDir)
	require.NoError(t, err)

	h := head.New(nil)
	db := &tsdb.DB{Head: h, Blocks: store}
	ts := int64(1_700_000_000_000)
	labels := model.FromMap("requests_total", map[string]string{"instance": "storage-a"})
	_, err = h.Append(labels, ts, 42)
	require.NoError(t, err)

	require.NoError(t, os.Remove(blocksDir))
	require.NoError(t, os.WriteFile(blocksDir, []byte("not a directory"), 0o600))

	b, err := db.Compact()
	require.Error(t, err)
	require.Nil(t, b)

	got, _ := db.Query(nil, ts, ts)
	require.Len(t, got, 1, "a failed compaction must leave the head queryable")
	require.Equal(t, labels, got[0].Labels)
	require.Equal(t, []model.Sample{{T: ts, V: 42}}, got[0].Samples)

	require.NoError(t, os.Remove(blocksDir))
	require.NoError(t, os.MkdirAll(blocksDir, 0o755))
	b, err = db.Compact()
	require.NoError(t, err)
	require.NotNil(t, b, "the samples must remain available for a later retry")

	got, _ = db.Query(nil, ts, ts)
	require.Len(t, got, 1)
	require.Equal(t, []model.Sample{{T: ts, V: 42}}, got[0].Samples)
}
