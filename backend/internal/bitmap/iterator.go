package bitmap

type Iter struct {
	ids []uint32
	i   int
}

func newIter(b *Bitmap) *Iter {
	return &Iter{ids: b.ToArray(), i: -1}
}

func (it *Iter) Next() bool {
	it.i++
	return it.i < len(it.ids)
}

func (it *Iter) Value() uint32 {
	return it.ids[it.i]
}
