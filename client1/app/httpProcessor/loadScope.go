package httpProcessor

import (
	"context"
	"sync"
)

// LoadScope belongs to a view, not the shared HTTP client. Starting a newer
// load or leaving the view cancels its fetch and invalidates late responses.
type LoadScope struct {
	mu         sync.Mutex
	generation uint64
	cancel     context.CancelFunc
	loading    bool
	err        error
}
type LoadTicket struct {
	scope      *LoadScope
	generation uint64
	Context    context.Context
}

func (s *LoadScope) Start() LoadTicket {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.generation++
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.loading = true
	s.err = nil
	return LoadTicket{s, s.generation, ctx}
}
func (s *LoadScope) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.loading = false
	s.err = nil
}

// Finish rejects stale results before the caller updates rows, pagination or DOM.
func (t LoadTicket) Finish(err error) bool {
	s := t.scope
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.generation != s.generation {
		return false
	}
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.loading = false
	s.err = err
	return true
}
func (s *LoadScope) State() (bool, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.loading, s.err }
