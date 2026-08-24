package downsample

import "github.com/alkaid/miniprometheus/internal/model"

func MinMax(ss []model.Sample, buckets int) []model.Sample {
	if buckets <= 0 || len(ss) <= buckets*2 {
		return ss
	}
	if len(ss) == 0 {
		return ss
	}
	minT, maxT := ss[0].T, ss[len(ss)-1].T
	span := maxT - minT
	if span <= 0 {
		return ss[:1]
	}
	type acc struct {
		min, max *model.Sample
	}
	arr := make([]acc, buckets)
	for i := range ss {
		b := int((ss[i].T - minT) * int64(buckets) / (span + 1))
		if b >= buckets {
			b = buckets - 1
		}
		if arr[b].min == nil || ss[i].V < arr[b].min.V {
			p := ss[i]
			arr[b].min = &p
		}
		if arr[b].max == nil || ss[i].V > arr[b].max.V {
			p := ss[i]
			arr[b].max = &p
		}
	}
	out := make([]model.Sample, 0, buckets*2)
	for _, a := range arr {
		if a.min == nil {
			continue
		}
		if a.min.T <= a.max.T {
			out = append(out, *a.min)
			if a.max.T != a.min.T {
				out = append(out, *a.max)
			}
		} else {
			out = append(out, *a.max)
			out = append(out, *a.min)
		}
	}
	return out
}

func Limit(ss []model.Sample, maxPoints int) []model.Sample {
	if maxPoints <= 0 || len(ss) <= maxPoints {
		return ss
	}
	return LTTB(ss, maxPoints)
}
