package encode

func WriteDoD(w *Writer, ts []int64) {
	if len(ts) == 0 {
		return
	}
	w.WriteBits(uint64(ts[0]), 64)
	var prevDelta int64
	for i := 1; i < len(ts); i++ {
		delta := ts[i] - ts[i-1]
		dod := delta - prevDelta
		prevDelta = delta
		writeDoD(w, dod)
	}
}

func writeDoD(w *Writer, dod int64) {
	switch {
	case dod == 0:
		w.WriteBit(0)
	case dod >= -63 && dod <= 64:
		w.WriteBits(0b10, 2)
		w.WriteBits(uint64(dod), 7)
	case dod >= -255 && dod <= 256:
		w.WriteBits(0b110, 3)
		w.WriteBits(uint64(dod), 9)
	case dod >= -2047 && dod <= 2048:
		w.WriteBits(0b1110, 4)
		w.WriteBits(uint64(dod), 12)
	case dod >= -2147483648 && dod <= 2147483647:
		w.WriteBits(0b1111, 4)
		w.WriteBits(uint64(int32(dod)), 32)
	default:
		w.WriteBits(0b1111, 4)
		w.WriteBits(0, 32)
		w.WriteBits(uint64(dod), 64)
	}
}

func ReadDoD(r *Reader, n int) ([]int64, bool) {
	if n <= 0 {
		return nil, true
	}
	out := make([]int64, n)
	t0, ok := r.ReadBits(64)
	if !ok {
		return nil, false
	}
	out[0] = int64(t0)
	var prevDelta int64
	for i := 1; i < n; i++ {
		dod, ok := readDoD(r)
		if !ok {
			return nil, false
		}
		delta := prevDelta + dod
		out[i] = out[i-1] + delta
		prevDelta = delta
	}
	return out, true
}

func readDoD(r *Reader) (int64, bool) {
	b, ok := r.ReadBit()
	if !ok {
		return 0, false
	}
	if b == 0 {
		return 0, true
	}
	b, ok = r.ReadBit()
	if !ok {
		return 0, false
	}
	if b == 0 {
		v, ok := r.ReadBits(7)
		return SignExtend(v, 7), ok
	}
	b, ok = r.ReadBit()
	if !ok {
		return 0, false
	}
	if b == 0 {
		v, ok := r.ReadBits(9)
		return SignExtend(v, 9), ok
	}
	b, ok = r.ReadBit()
	if !ok {
		return 0, false
	}
	if b == 0 {
		v, ok := r.ReadBits(12)
		return SignExtend(v, 12), ok
	}
	v, ok := r.ReadBits(32)
	if !ok {
		return 0, false
	}
	if v == 0 {
		v64, ok := r.ReadBits(64)
		return int64(v64), ok
	}
	return SignExtend(v, 32), true
}
