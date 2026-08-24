package downsample

import "github.com/alkaid/miniprometheus/internal/model"

func LTTB(ss []model.Sample, threshold int) []model.Sample {
	n := len(ss)
	if threshold <= 0 || n <= threshold {
		return ss
	}
	if threshold == 1 {
		return []model.Sample{ss[0]}
	}
	out := make([]model.Sample, 0, threshold)
	out = append(out, ss[0])
	bucketSize := float64(n-2) / float64(threshold-2)
	a := 0
	for i := 0; i < threshold-2; i++ {
		avgStart := int(float64(i+1)*bucketSize) + 1
		avgEnd := int(float64(i+2)*bucketSize) + 1
		if avgEnd >= n {
			avgEnd = n
		}
		if avgStart >= avgEnd {
			avgStart = avgEnd - 1
		}
		var avgT, avgV float64
		for j := avgStart; j < avgEnd; j++ {
			avgT += float64(ss[j].T)
			avgV += ss[j].V
		}
		cnt := float64(avgEnd - avgStart)
		avgT /= cnt
		avgV /= cnt
		rangeStart := int(float64(i)*bucketSize) + 1
		rangeEnd := int(float64(i+1)*bucketSize) + 1
		if rangeEnd >= n-1 {
			rangeEnd = n - 1
		}
		if rangeStart >= rangeEnd {
			rangeStart = rangeEnd - 1
		}
		ax, ay := float64(ss[a].T), ss[a].V
		maxArea := -1.0
		next := rangeStart
		for j := rangeStart; j < rangeEnd; j++ {
			area := areaOf(ax, ay, float64(ss[j].T), ss[j].V, avgT, avgV)
			if area > maxArea {
				maxArea = area
				next = j
			}
		}
		out = append(out, ss[next])
		a = next
	}
	out = append(out, ss[n-1])
	return out
}

func areaOf(ax, ay, bx, by, cx, cy float64) float64 {
	v := (ax-cx)*(by-ay) - (ax-bx)*(cy-ay)
	if v < 0 {
		return -v
	}
	return v
}
