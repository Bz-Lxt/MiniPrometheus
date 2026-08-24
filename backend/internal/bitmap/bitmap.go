package bitmap

import "sort"

type Bitmap struct {
	keys []uint16
	cons []*container
}

func New() *Bitmap { return &Bitmap{} }

func FromIDs(ids []uint32) *Bitmap {
	b := New()
	for _, id := range ids {
		b.Add(id)
	}
	return b
}

func (b *Bitmap) clone() *Bitmap {
	o := &Bitmap{keys: append([]uint16(nil), b.keys...), cons: make([]*container, len(b.cons))}
	for i, c := range b.cons {
		o.cons[i] = c.clone()
	}
	return o
}

func (b *Bitmap) find(key uint16) (int, bool) {
	i := sort.Search(len(b.keys), func(i int) bool { return b.keys[i] >= key })
	if i < len(b.keys) && b.keys[i] == key {
		return i, true
	}
	return i, false
}

func (b *Bitmap) Add(id uint32) {
	key := uint16(id >> 16)
	val := uint16(id)
	i, ok := b.find(key)
	if !ok {
		b.keys = append(b.keys, 0)
		b.cons = append(b.cons, nil)
		copy(b.keys[i+1:], b.keys[i:])
		copy(b.cons[i+1:], b.cons[i:])
		b.keys[i] = key
		b.cons[i] = newArray(nil)
	}
	b.cons[i].add(val)
}

func (b *Bitmap) Contains(id uint32) bool {
	i, ok := b.find(uint16(id >> 16))
	if !ok {
		return false
	}
	return b.cons[i].contains(uint16(id))
}

func (b *Bitmap) Cardinality() int {
	n := 0
	for _, c := range b.cons {
		n += c.card
	}
	return n
}

func (b *Bitmap) And(o *Bitmap) *Bitmap {
	if b == nil || o == nil {
		return New()
	}
	out := New()
	i, j := 0, 0
	for i < len(b.keys) && j < len(o.keys) {
		if b.keys[i] == o.keys[j] {
			c := andContainer(b.cons[i], o.cons[j])
			if c.card > 0 {
				out.keys = append(out.keys, b.keys[i])
				out.cons = append(out.cons, c)
			}
			i++
			j++
		} else if b.keys[i] < o.keys[j] {
			i++
		} else {
			j++
		}
	}
	return out
}

func (b *Bitmap) Or(o *Bitmap) *Bitmap {
	if b == nil || b.Cardinality() == 0 {
		if o == nil {
			return New()
		}
		return o.clone()
	}
	if o == nil || o.Cardinality() == 0 {
		return b.clone()
	}
	out := New()
	i, j := 0, 0
	for i < len(b.keys) || j < len(o.keys) {
		switch {
		case j == len(o.keys) || (i < len(b.keys) && b.keys[i] < o.keys[j]):
			out.keys = append(out.keys, b.keys[i])
			out.cons = append(out.cons, b.cons[i].clone())
			i++
		case i == len(b.keys) || (j < len(o.keys) && o.keys[j] < b.keys[i]):
			out.keys = append(out.keys, o.keys[j])
			out.cons = append(out.cons, o.cons[j].clone())
			j++
		default:
			out.keys = append(out.keys, b.keys[i])
			out.cons = append(out.cons, orContainer(b.cons[i], o.cons[j]))
			i++
			j++
		}
	}
	return out
}

func (b *Bitmap) AndNot(o *Bitmap) *Bitmap {
	if b == nil {
		return New()
	}
	if o == nil || o.Cardinality() == 0 {
		return b.clone()
	}
	out := New()
	i, j := 0, 0
	for i < len(b.keys) {
		if j == len(o.keys) || b.keys[i] < o.keys[j] {
			out.keys = append(out.keys, b.keys[i])
			out.cons = append(out.cons, b.cons[i].clone())
			i++
			continue
		}
		if o.keys[j] < b.keys[i] {
			j++
			continue
		}
		c := andNotContainer(b.cons[i], o.cons[j])
		if c.card > 0 {
			out.keys = append(out.keys, b.keys[i])
			out.cons = append(out.cons, c)
		}
		i++
		j++
	}
	return out
}

func (b *Bitmap) ToArray() []uint32 {
	if b == nil || b.Cardinality() == 0 {
		return nil
	}
	out := make([]uint32, 0, b.Cardinality())
	scratch := make([]uint16, 0, 64)
	for i, c := range b.cons {
		scratch = scratch[:0]
		scratch = c.values(scratch)
		base := uint32(b.keys[i]) << 16
		for _, v := range scratch {
			out = append(out, base|uint32(v))
		}
	}
	return out
}

func (b *Bitmap) Iterator() *Iter {
	return newIter(b)
}
