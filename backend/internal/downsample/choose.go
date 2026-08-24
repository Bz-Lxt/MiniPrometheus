package downsample

import "github.com/alkaid/miniprometheus/internal/model"

func ForPixels(ss []model.Sample, pixelWidth int) []model.Sample {
	if pixelWidth <= 0 {
		return ss
	}
	limit := pixelWidth * 4
	if len(ss) <= limit {
		return ss
	}
	return LTTB(ss, limit)
}
