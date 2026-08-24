package encode

import (
	"math"
	"math/bits"
)

func WriteGorilla(w *Writer, vs []float64) {
	if len(vs) == 0 {
		return
	}
	prev := math.Float64bits(vs[0])
	w.WriteBits(prev, 64)
	var lead, trail int
	first := true
	for i := 1; i < len(vs); i++ {
		cur := math.Float64bits(vs[i])
		xor := prev ^ cur
		prev = cur
		if xor == 0 {
			w.WriteBit(0)
			continue
		}
		w.WriteBit(1)
		lz := bits.LeadingZeros64(xor)
		tz := bits.TrailingZeros64(xor)
		if lz > 31 {
			lz = 31
		}
		if !first && lz >= lead && tz >= trail {
			w.WriteBit(0)
			mbits := 64 - lead - trail
			w.WriteBits(xor>>uint(trail), mbits)
			continue
		}
		w.WriteBit(1)
		mbits := 64 - lz - tz
		if mbits <= 0 {
			mbits = 1
			tz = 64 - lz - mbits
		}
		w.WriteBits(uint64(lz), 5)
		w.WriteBits(uint64(mbits-1), 6)
		w.WriteBits(xor>>uint(tz), mbits)
		lead, trail = lz, tz
		first = false
	}
}

func ReadGorilla(r *Reader, n int) ([]float64, bool) {
	if n <= 0 {
		return nil, true
	}
	out := make([]float64, n)
	b0, ok := r.ReadBits(64)
	if !ok {
		return nil, false
	}
	prev := b0
	out[0] = math.Float64frombits(prev)
	var lead, trail int
	first := true
	for i := 1; i < n; i++ {
		ctl, ok := r.ReadBit()
		if !ok {
			return nil, false
		}
		if ctl == 0 {
			out[i] = math.Float64frombits(prev)
			continue
		}
		reuse, ok := r.ReadBit()
		if !ok {
			return nil, false
		}
		var xor uint64
		if reuse == 0 && !first {
			mbits := 64 - lead - trail
			v, ok := r.ReadBits(mbits)
			if !ok {
				return nil, false
			}
			xor = v << uint(trail)
		} else {
			lz, ok := r.ReadBits(5)
			if !ok {
				return nil, false
			}
			mlen, ok := r.ReadBits(6)
			if !ok {
				return nil, false
			}
			mbits := int(mlen) + 1
			v, ok := r.ReadBits(mbits)
			if !ok {
				return nil, false
			}
			lead = int(lz)
			trail = 64 - lead - mbits
			if trail < 0 {
				trail = 0
			}
			xor = v << uint(trail)
			first = false
		}
		prev ^= xor
		out[i] = math.Float64frombits(prev)
	}
	return out, true
}
