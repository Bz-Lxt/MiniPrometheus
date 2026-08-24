package model

import (
	"hash/fnv"
	"sort"
	"strings"
)

const MetricName = "__name__"

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Labels []Label

func (ls Labels) Len() int           { return len(ls) }
func (ls Labels) Swap(i, j int)      { ls[i], ls[j] = ls[j], ls[i] }
func (ls Labels) Less(i, j int) bool { return ls[i].Name < ls[j].Name }

func Normalize(ls Labels) Labels {
	cp := make(Labels, 0, len(ls))
	seen := make(map[string]int, len(ls))
	for _, l := range ls {
		l.Name = strings.TrimSpace(l.Name)
		if l.Name == "" {
			continue
		}
		if i, ok := seen[l.Name]; ok {
			cp[i].Value = l.Value
			continue
		}
		seen[l.Name] = len(cp)
		cp = append(cp, l)
	}
	sort.Sort(cp)
	return cp
}

func (ls Labels) Get(name string) string {
	for _, l := range ls {
		if l.Name == name {
			return l.Value
		}
	}
	return ""
}

func (ls Labels) Map() map[string]string {
	m := make(map[string]string, len(ls))
	for _, l := range ls {
		m[l.Name] = l.Value
	}
	return m
}

func (ls Labels) Without(names ...string) Labels {
	drop := make(map[string]struct{}, len(names))
	for _, n := range names {
		drop[n] = struct{}{}
	}
	out := make(Labels, 0, len(ls))
	for _, l := range ls {
		if _, ok := drop[l.Name]; !ok {
			out = append(out, l)
		}
	}
	return out
}

func (ls Labels) Keep(names ...string) Labels {
	keep := make(map[string]struct{}, len(names))
	for _, n := range names {
		keep[n] = struct{}{}
	}
	out := make(Labels, 0, len(names))
	for _, l := range ls {
		if _, ok := keep[l.Name]; ok {
			out = append(out, l)
		}
	}
	return Normalize(out)
}

func (ls Labels) Hash() uint64 {
	h := fnv.New64a()
	for _, l := range ls {
		_, _ = h.Write([]byte(l.Name))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(l.Value))
		_, _ = h.Write([]byte{255})
	}
	return h.Sum64()
}

func (ls Labels) Equal(o Labels) bool {
	if len(ls) != len(o) {
		return false
	}
	for i := range ls {
		if ls[i].Name != o[i].Name || ls[i].Value != o[i].Value {
			return false
		}
	}
	return true
}

func (ls Labels) String() string {
	var b strings.Builder
	name := ls.Get(MetricName)
	b.WriteString(name)
	b.WriteByte('{')
	first := true
	for _, l := range ls {
		if l.Name == MetricName {
			continue
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(l.Name)
		b.WriteString("=\"")
		b.WriteString(l.Value)
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

func FromMap(metric string, m map[string]string) Labels {
	ls := make(Labels, 0, len(m)+1)
	if metric != "" {
		ls = append(ls, Label{Name: MetricName, Value: metric})
	}
	for k, v := range m {
		if k == MetricName {
			continue
		}
		ls = append(ls, Label{Name: k, Value: v})
	}
	return Normalize(ls)
}
