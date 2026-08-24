package index

import (
	"strings"
	"sync"
	"time"

	"github.com/alkaid/miniprometheus/internal/bitmap"
	"github.com/alkaid/miniprometheus/internal/model"
)

type Inverted struct {
	mu      sync.RWMutex
	all     *bitmap.Bitmap
	post    map[string]map[string]*bitmap.Bitmap
	names   map[string]*bitmap.Bitmap
	series  map[model.SeriesID]model.Labels
	hash    map[uint64][]model.SeriesID
	nextID  model.SeriesID
	symbols *Symbols
}

func NewInverted() *Inverted {
	return &Inverted{
		all:     bitmap.New(),
		post:    make(map[string]map[string]*bitmap.Bitmap),
		names:   make(map[string]*bitmap.Bitmap),
		series:  make(map[model.SeriesID]model.Labels),
		hash:    make(map[uint64][]model.SeriesID),
		nextID:  1,
		symbols: NewSymbols(),
	}
}

func (iv *Inverted) GetOrCreate(ls model.Labels) (model.SeriesID, bool) {
	ls = model.Normalize(ls)
	h := ls.Hash()
	iv.mu.Lock()
	defer iv.mu.Unlock()
	if ids, ok := iv.hash[h]; ok {
		for _, id := range ids {
			if iv.series[id].Equal(ls) {
				return id, false
			}
		}
	}
	id := iv.nextID
	iv.nextID++
	cp := append(model.Labels(nil), ls...)
	iv.series[id] = cp
	iv.hash[h] = append(iv.hash[h], id)
	iv.all.Add(uint32(id))
	for _, l := range cp {
		iv.symbols.Intern(l.Name)
		iv.symbols.Intern(l.Value)
		nm := iv.post[l.Name]
		if nm == nil {
			nm = make(map[string]*bitmap.Bitmap)
			iv.post[l.Name] = nm
		}
		bm := nm[l.Value]
		if bm == nil {
			bm = bitmap.New()
			nm[l.Value] = bm
		}
		bm.Add(uint32(id))
		nb := iv.names[l.Name]
		if nb == nil {
			nb = bitmap.New()
			iv.names[l.Name] = nb
		}
		nb.Add(uint32(id))
	}
	return id, true
}

func (iv *Inverted) Labels(id model.SeriesID) (model.Labels, bool) {
	iv.mu.RLock()
	defer iv.mu.RUnlock()
	ls, ok := iv.series[id]
	return ls, ok
}

func (iv *Inverted) SeriesCount() int {
	iv.mu.RLock()
	defer iv.mu.RUnlock()
	return len(iv.series)
}

func (iv *Inverted) LabelNames() []string {
	iv.mu.RLock()
	defer iv.mu.RUnlock()
	out := make([]string, 0, len(iv.post))
	for n := range iv.post {
		out = append(out, n)
	}
	return out
}

func (iv *Inverted) LabelValues(name string) []string {
	iv.mu.RLock()
	defer iv.mu.RUnlock()
	nm := iv.post[name]
	out := make([]string, 0, len(nm))
	for v := range nm {
		out = append(out, v)
	}
	return out
}

type LookupStat struct {
	DurationUS int64 `json:"duration_us"`
	Series     int   `json:"series"`
	HitIndex   bool  `json:"hit_index"`
}

func (iv *Inverted) Restore(id model.SeriesID, ls model.Labels) {
	ls = model.Normalize(ls)
	h := ls.Hash()
	iv.mu.Lock()
	defer iv.mu.Unlock()
	if _, ok := iv.series[id]; ok {
		return
	}
	iv.series[id] = append(model.Labels(nil), ls...)
	iv.hash[h] = append(iv.hash[h], id)
	iv.all.Add(uint32(id))
	if id >= iv.nextID {
		iv.nextID = id + 1
	}
	for _, l := range ls {
		iv.symbols.Intern(l.Name)
		iv.symbols.Intern(l.Value)
		nm := iv.post[l.Name]
		if nm == nil {
			nm = make(map[string]*bitmap.Bitmap)
			iv.post[l.Name] = nm
		}
		bm := nm[l.Value]
		if bm == nil {
			bm = bitmap.New()
			nm[l.Value] = bm
		}
		bm.Add(uint32(id))
		nb := iv.names[l.Name]
		if nb == nil {
			nb = bitmap.New()
			iv.names[l.Name] = nb
		}
		nb.Add(uint32(id))
	}
}

func (iv *Inverted) Lookup(ms []*Matcher) (*bitmap.Bitmap, LookupStat) {
	start := time.Now()
	st := LookupStat{HitIndex: true}
	if err := compileAll(ms); err != nil {
		return bitmap.New(), st
	}
	iv.mu.RLock()
	defer iv.mu.RUnlock()
	var acc *bitmap.Bitmap
	for _, m := range ms {
		set := iv.matchOne(m)
		if m.Negative() {
			if acc == nil {
				acc = iv.all.AndNot(set)
			} else {
				acc = acc.AndNot(set)
			}
			continue
		}
		if acc == nil {
			acc = set
		} else {
			acc = acc.And(set)
		}
	}
	if acc == nil {
		acc = iv.all
	}
	st.Series = acc.Cardinality()
	st.DurationUS = time.Since(start).Microseconds()
	return acc, st
}

func compileAll(ms []*Matcher) error {
	for _, m := range ms {
		if err := m.Compile(); err != nil {
			return err
		}
	}
	return nil
}

func (iv *Inverted) matchOne(m *Matcher) *bitmap.Bitmap {
	switch m.Type {
	case MatchEqual:
		if nm := iv.post[m.Name]; nm != nil {
			if bm := nm[m.Value]; bm != nil {
				return bm
			}
		}
		return bitmap.New()
	case MatchNotEqual:
		return iv.valuesMatching(m.Name, func(v string) bool { return v == m.Value })
	case MatchRegexp:
		pref := LiteralPrefix(m.Value)
		return iv.valuesMatching(m.Name, func(v string) bool {
			if pref != "" && !strings.HasPrefix(v, pref) {
				return false
			}
			return m.Matches(v)
		})
	case MatchNotRegexp:
		return iv.valuesMatching(m.Name, func(v string) bool { return m.re != nil && m.re.MatchString(v) })
	default:
		return bitmap.New()
	}
}

func (iv *Inverted) valuesMatching(name string, pred func(string) bool) *bitmap.Bitmap {
	nm := iv.post[name]
	if nm == nil {
		return bitmap.New()
	}
	var acc *bitmap.Bitmap
	for v, bm := range nm {
		if pred(v) {
			if acc == nil {
				acc = bm
			} else {
				acc = acc.Or(bm)
			}
		}
	}
	if acc == nil {
		return bitmap.New()
	}
	return acc
}

func (iv *Inverted) AllIDs() []model.SeriesID {
	iv.mu.RLock()
	defer iv.mu.RUnlock()
	out := make([]model.SeriesID, 0, len(iv.series))
	for id := range iv.series {
		out = append(out, id)
	}
	return out
}

func (iv *Inverted) Symbols() *Symbols { return iv.symbols }
