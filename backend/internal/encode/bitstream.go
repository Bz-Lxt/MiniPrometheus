package encode

type Writer struct {
	buf []byte
	bit uint8
}

func NewWriter(capHint int) *Writer {
	if capHint < 16 {
		capHint = 16
	}
	return &Writer{buf: make([]byte, 0, capHint)}
}

func (w *Writer) WriteBit(b uint8) {
	if w.bit == 0 {
		w.buf = append(w.buf, 0)
	}
	if b != 0 {
		w.buf[len(w.buf)-1] |= 1 << (7 - w.bit)
	}
	w.bit++
	if w.bit == 8 {
		w.bit = 0
	}
}

func (w *Writer) WriteBits(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.WriteBit(uint8((v >> uint(i)) & 1))
	}
}

func (w *Writer) Bytes() []byte {
	out := make([]byte, len(w.buf))
	copy(out, w.buf)
	return out
}

func (w *Writer) BitLen() int {
	if w.bit == 0 {
		return len(w.buf) * 8
	}
	return (len(w.buf)-1)*8 + int(w.bit)
}

func (w *Writer) Reset() {
	w.buf = w.buf[:0]
	w.bit = 0
}

type Reader struct {
	buf []byte
	pos int
	bit uint8
}

func NewReader(b []byte) *Reader {
	return &Reader{buf: b}
}

func (r *Reader) ReadBit() (uint8, bool) {
	if r.pos >= len(r.buf) {
		return 0, false
	}
	b := (r.buf[r.pos] >> (7 - r.bit)) & 1
	r.bit++
	if r.bit == 8 {
		r.bit = 0
		r.pos++
	}
	return b, true
}

func (r *Reader) ReadBits(n int) (uint64, bool) {
	var v uint64
	for i := 0; i < n; i++ {
		b, ok := r.ReadBit()
		if !ok {
			return 0, false
		}
		v = (v << 1) | uint64(b)
	}
	return v, true
}

func SignExtend(v uint64, width int) int64 {
	if width <= 0 || width >= 64 {
		return int64(v)
	}
	shift := 64 - width
	return int64(v<<shift) >> shift
}
