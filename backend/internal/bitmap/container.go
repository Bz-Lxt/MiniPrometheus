package bitmap

const arrayLimit = 4096

type kind uint8

const (
	kindArray kind = iota
	kindBitmap
)

type container struct {
	kind kind
	arr  []uint16
	bits []uint64
	card int
}

func newArray(vals []uint16) *container {
	cp := append([]uint16(nil), vals...)
	return &container{kind: kindArray, arr: cp, card: len(cp)}
}

func newBitmapFromArr(vals []uint16) *container {
	c := &container{kind: kindBitmap, bits: make([]uint64, 1024), card: len(vals)}
	for _, v := range vals {
		c.bits[v>>6] |= 1 << (v & 63)
	}
	return c
}

func (c *container) clone() *container {
	o := &container{kind: c.kind, card: c.card}
	if c.kind == kindArray {
		o.arr = append([]uint16(nil), c.arr...)
	} else {
		o.bits = append([]uint64(nil), c.bits...)
	}
	return o
}

func (c *container) add(v uint16) bool {
	if c.kind == kindArray {
		i, ok := search16(c.arr, v)
		if ok {
			return false
		}
		c.arr = append(c.arr, 0)
		copy(c.arr[i+1:], c.arr[i:])
		c.arr[i] = v
		c.card++
		if c.card > arrayLimit {
			*c = *newBitmapFromArr(c.arr)
		}
		return true
	}
	word, bit := v>>6, uint16(v&63)
	mask := uint64(1) << bit
	if c.bits[word]&mask != 0 {
		return false
	}
	c.bits[word] |= mask
	c.card++
	return true
}

func (c *container) contains(v uint16) bool {
	if c.kind == kindArray {
		_, ok := search16(c.arr, v)
		return ok
	}
	return c.bits[v>>6]&(1<<(v&63)) != 0
}

func search16(a []uint16, v uint16) (int, bool) {
	lo, hi := 0, len(a)
	for lo < hi {
		mid := (lo + hi) / 2
		if a[mid] < v {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(a) && a[lo] == v {
		return lo, true
	}
	return lo, false
}

func andContainer(a, b *container) *container {
	if a.kind == kindArray && b.kind == kindArray {
		return andArrArr(a.arr, b.arr)
	}
	if a.kind == kindArray {
		return andArrBits(a.arr, b.bits)
	}
	if b.kind == kindArray {
		return andArrBits(b.arr, a.bits)
	}
	return andBitsBits(a.bits, b.bits)
}

func orContainer(a, b *container) *container {
	if a.kind == kindArray && b.kind == kindArray {
		return orArrArr(a.arr, b.arr)
	}
	if a.kind == kindBitmap && b.kind == kindBitmap {
		return orBitsBits(a.bits, b.bits)
	}
	if a.kind == kindArray {
		o := b.clone()
		for _, v := range a.arr {
			o.add(v)
		}
		return o
	}
	o := a.clone()
	for _, v := range b.arr {
		o.add(v)
	}
	return o
}

func andNotContainer(a, b *container) *container {
	if a.kind == kindArray {
		out := make([]uint16, 0, len(a.arr))
		for _, v := range a.arr {
			if !b.contains(v) {
				out = append(out, v)
			}
		}
		return newArray(out)
	}
	if b.kind == kindArray {
		o := a.clone()
		for _, v := range b.arr {
			word, bit := v>>6, uint16(v&63)
			mask := uint64(1) << bit
			if o.bits[word]&mask != 0 {
				o.bits[word] &^= mask
				o.card--
			}
		}
		return o
	}
	bits := make([]uint64, 1024)
	card := 0
	for i := 0; i < 1024; i++ {
		w := a.bits[i] &^ b.bits[i]
		bits[i] = w
		card += popcnt(w)
	}
	return bitsToContainer(bits, card)
}

func andArrArr(a, b []uint16) *container {
	out := make([]uint16, 0, min(len(a), len(b)))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			out = append(out, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}
	return newArray(out)
}

func orArrArr(a, b []uint16) *container {
	out := make([]uint16, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			out = append(out, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	out = append(out, b[j:]...)
	if len(out) > arrayLimit {
		return newBitmapFromArr(out)
	}
	return newArray(out)
}

func andArrBits(arr []uint16, bits []uint64) *container {
	out := make([]uint16, 0, len(arr))
	for _, v := range arr {
		if bits[v>>6]&(1<<(v&63)) != 0 {
			out = append(out, v)
		}
	}
	return newArray(out)
}

func andBitsBits(a, b []uint64) *container {
	bits := make([]uint64, 1024)
	card := 0
	for i := 0; i < 1024; i++ {
		w := a[i] & b[i]
		bits[i] = w
		card += popcnt(w)
	}
	return bitsToContainer(bits, card)
}

func orBitsBits(a, b []uint64) *container {
	bits := make([]uint64, 1024)
	card := 0
	for i := 0; i < 1024; i++ {
		w := a[i] | b[i]
		bits[i] = w
		card += popcnt(w)
	}
	return bitsToContainer(bits, card)
}

func bitsToContainer(bits []uint64, card int) *container {
	if card <= arrayLimit {
		arr := make([]uint16, 0, card)
		for i, w := range bits {
			for w != 0 {
				t := w & -w
				bit := trailing(w)
				arr = append(arr, uint16(i<<6+bit))
				w ^= t
			}
		}
		return newArray(arr)
	}
	return &container{kind: kindBitmap, bits: bits, card: card}
}

func popcnt(x uint64) int {
	x = x - ((x >> 1) & 0x5555555555555555)
	x = (x & 0x3333333333333333) + ((x >> 2) & 0x3333333333333333)
	x = (x + (x >> 4)) & 0x0f0f0f0f0f0f0f0f
	return int((x * 0x0101010101010101) >> 56)
}

func trailing(x uint64) int {
	if x == 0 {
		return 64
	}
	n := 0
	if x&0xffffffff == 0 {
		n += 32
		x >>= 32
	}
	if x&0xffff == 0 {
		n += 16
		x >>= 16
	}
	if x&0xff == 0 {
		n += 8
		x >>= 8
	}
	if x&0xf == 0 {
		n += 4
		x >>= 4
	}
	if x&0x3 == 0 {
		n += 2
		x >>= 2
	}
	if x&0x1 == 0 {
		n++
	}
	return n
}

func (c *container) values(dst []uint16) []uint16 {
	if c.kind == kindArray {
		return append(dst, c.arr...)
	}
	for i, w := range c.bits {
		for w != 0 {
			t := w & -w
			bit := trailing(w)
			dst = append(dst, uint16(i<<6+bit))
			w ^= t
		}
	}
	return dst
}
