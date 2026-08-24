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
