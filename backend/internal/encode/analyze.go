package encode

import (
	"math"
	"math/bits"
)

type BranchStats struct {
	DoDZero   int `json:"dod_zero"`
	DoD7      int `json:"dod_7"`
	DoD9      int `json:"dod_9"`
	DoD12     int `json:"dod_12"`
	DoD32     int `json:"dod_32"`
	DoD64     int `json:"dod_64"`
	XORZero   int `json:"xor_zero"`
	XORReuse  int `json:"xor_reuse"`
	XORReset  int `json:"xor_reset"`
	Points    int `json:"points"`
	TSBits    int `json:"ts_bits"`
	ValBits   int `json:"val_bits"`
}

func Analyze(ts []int64, vs []float64) BranchStats {
	st := BranchStats{Points: len(ts)}
	if len(ts) == 0 {
		return st
	}
	st.TSBits += 64
	var prevDelta int64
	for i := 1; i < len(ts); i++ {
		delta := ts[i] - ts[i-1]
		dod := delta - prevDelta
		prevDelta = delta
		switch {
		case dod == 0:
			st.DoDZero++
			st.TSBits++
		case dod >= -63 && dod <= 64:
			st.DoD7++
			st.TSBits += 9
		case dod >= -255 && dod <= 256:
			st.DoD9++
			st.TSBits += 12
		case dod >= -2047 && dod <= 2048:
			st.DoD12++
			st.TSBits += 16
		case dod >= -2147483648 && dod <= 2147483647:
			st.DoD32++
			st.TSBits += 36
		default:
			st.DoD64++
			st.TSBits += 100
		}
	}
	if len(vs) == 0 {
		return st
	}
	st.ValBits += 64
	prev := math.Float64bits(vs[0])
	var lead, trail int
	first := true
	for i := 1; i < len(vs); i++ {
		cur := math.Float64bits(vs[i])
		xor := prev ^ cur
		prev = cur
		if xor == 0 {
			st.XORZero++
			st.ValBits++
			continue
		}
		lz := bits.LeadingZeros64(xor)
		tz := bits.TrailingZeros64(xor)
		if lz > 31 {
			lz = 31
		}
		if !first && lz >= lead && tz >= trail {
			st.XORReuse++
			st.ValBits += 2 + (64 - lead - trail)
			continue
		}
		mbits := 64 - lz - tz
		if mbits <= 0 {
			mbits = 1
		}
		st.XORReset++
		st.ValBits += 2 + 5 + 6 + mbits
		lead, trail = lz, tz
		first = false
	}
	return st
}

func (s BranchStats) BytesPerSample() float64 {
	if s.Points == 0 {
		return 0
	}
	return float64(s.TSBits+s.ValBits) / 8.0 / float64(s.Points)
}
