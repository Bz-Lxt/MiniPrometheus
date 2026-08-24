package index

import (
	"github.com/alkaid/miniprometheus/internal/bitmap"
	"github.com/alkaid/miniprometheus/internal/model"
)

func IDsFromBitmap(b *bitmap.Bitmap) []model.SeriesID {
	raw := b.ToArray()
	out := make([]model.SeriesID, len(raw))
	for i, v := range raw {
		out[i] = model.SeriesID(v)
	}
	return out
}
