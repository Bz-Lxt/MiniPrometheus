package index

import "sync"

type Symbols struct {
	mu   sync.RWMutex
	ids  map[string]uint32
	strs []string
}

func NewSymbols() *Symbols {
	return &Symbols{ids: make(map[string]uint32), strs: []string{""}}
}

func (s *Symbols) Intern(v string) uint32 {
	s.mu.RLock()
	if id, ok := s.ids[v]; ok {
		s.mu.RUnlock()
		return id
	}
	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.ids[v]; ok {
		return id
	}
	id := uint32(len(s.strs))
	s.strs = append(s.strs, v)
	s.ids[v] = id
	return id
}

func (s *Symbols) Lookup(id uint32) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if int(id) >= len(s.strs) {
		return ""
	}
	return s.strs[id]
}

func (s *Symbols) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.strs) - 1
}
