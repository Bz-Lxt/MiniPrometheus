package model

import "math"

func CompareSamples(a, b []Sample) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].T != b[i].T {
			return false
		}
		if !sameBits(a[i].V, b[i].V) {
			return false
		}
	}
	return true
}

func sameBits(a, b float64) bool {
	return math.Float64bits(a) == math.Float64bits(b)
}
