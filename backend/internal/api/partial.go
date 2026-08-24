package api

func (s *Server) Degraded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.degraded...)
}

func (s *Server) MarkDegraded(eps ...string) {
	s.mu.Lock()
	s.degraded = append([]string(nil), eps...)
	s.mu.Unlock()
}

func (s *Server) ClearDegraded() {
	s.mu.Lock()
	s.degraded = nil
	s.mu.Unlock()
}
