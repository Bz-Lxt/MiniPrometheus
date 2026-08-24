package bitmap

import "sort"

type Sorted struct {
	ids []uint32
}

func NewSorted(ids []uint32) *Sorted {
	cp := append([]uint32(nil), ids...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	out := cp[:0]
	var last uint32
	for i, id := range cp {
		if i == 0 || id != last {
			out = append(out, id)
			last = id
		}
	}
	return &Sorted{ids: out}
}

func (s *Sorted) And(o *Sorted) *Sorted {
	out := make([]uint32, 0, min(len(s.ids), len(o.ids)))
	i, j := 0, 0
	for i < len(s.ids) && j < len(o.ids) {
		if s.ids[i] == o.ids[j] {
			out = append(out, s.ids[i])
			i++
			j++
		} else if s.ids[i] < o.ids[j] {
			i++
		} else {
			j++
		}
	}
	return &Sorted{ids: out}
}

func (s *Sorted) Or(o *Sorted) *Sorted {
	out := make([]uint32, 0, len(s.ids)+len(o.ids))
	i, j := 0, 0
	for i < len(s.ids) && j < len(o.ids) {
		if s.ids[i] == o.ids[j] {
			out = append(out, s.ids[i])
			i++
			j++
		} else if s.ids[i] < o.ids[j] {
			out = append(out, s.ids[i])
			i++
		} else {
			out = append(out, o.ids[j])
			j++
		}
	}
	out = append(out, s.ids[i:]...)
	out = append(out, o.ids[j:]...)
	return &Sorted{ids: out}
}

func (s *Sorted) AndNot(o *Sorted) *Sorted {
	out := make([]uint32, 0, len(s.ids))
	i, j := 0, 0
	for i < len(s.ids) {
		if j >= len(o.ids) || s.ids[i] < o.ids[j] {
			out = append(out, s.ids[i])
			i++
			continue
		}
		if s.ids[i] == o.ids[j] {
			i++
			j++
			continue
		}
		j++
	}
	return &Sorted{ids: out}
}

func (s *Sorted) IDs() []uint32 { return append([]uint32(nil), s.ids...) }
