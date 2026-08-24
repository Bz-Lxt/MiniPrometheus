package encode

import "math"

func XORZeroRatio(vs []float64) float64 {
	if len(vs) < 2 {
		return 0
	}
	z := 0
	prev := math.Float64bits(vs[0])
	for i := 1; i < len(vs); i++ {
		cur := math.Float64bits(vs[i])
		if prev^cur == 0 {
			z++
		}
		prev = cur
	}
	return float64(z) / float64(len(vs)-1)
}

func RegularInterval(ts []int64) bool {
	if len(ts) < 3 {
		return true
	}
	d := ts[1] - ts[0]
	for i := 2; i < len(ts); i++ {
		if ts[i]-ts[i-1] != d {
			return false
		}
	}
	return true
}
