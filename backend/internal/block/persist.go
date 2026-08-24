package block

import (
	"github.com/alkaid/miniprometheus/internal/head"
	"github.com/alkaid/miniprometheus/internal/model"
)

func CompactHead(h *head.Head, store *Store) (*Block, error) {
	frozen := h.SnapshotSealed()
	if len(frozen) == 0 {
		return nil, nil
	}
	b, err := Persist(store.dir, frozen)
	if err != nil {
		// Do NOT drop sealed chunks on failure: they must stay in the Head
		// so they remain queryable and can be retried on the next compaction.
		// Dropping them here would cause already-received samples to vanish
		// from both online queries and subsequent retries.
		return nil, err
	}
	ids := make([]model.SeriesID, 0, len(frozen))
	for _, f := range frozen {
		ids = append(ids, f.ID)
	}
	h.DropSealed(ids)
	store.Add(b)
	return b, nil
}
