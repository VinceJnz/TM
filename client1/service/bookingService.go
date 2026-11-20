package service

import (
	"client1/v2/app/api"
	"sync"
	"time"
)

type Booking = api.BookingDTO

// BookingService owns cached state; the API adapter owns transport and DTOs.
type BookingService struct {
	mu    sync.Mutex
	state DataState[[]Booking]
	ttl   time.Duration
	api   api.BookingAPI
}

func NewBookingService(adapter api.BookingAPI) *BookingService {
	return &BookingService{api: adapter, ttl: time.Minute}
}
func (s *BookingService) GetState() DataState[[]Booking] {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state
	state.Data = append([]Booking(nil), state.Data...)
	return state
}
func (s *BookingService) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.LastFetched = time.Time{}
}
func (s *BookingService) LoadIfNeeded() {
	s.mu.Lock()
	if s.state.Loading || (!s.state.LastFetched.IsZero() && time.Since(s.state.LastFetched) < s.ttl) {
		s.mu.Unlock()
		return
	}
	s.state.Loading = true
	s.state.Err = nil
	s.mu.Unlock()
	go s.api.GetBookings(func(data []Booking) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.state.Data = append([]Booking(nil), data...)
		s.state.LastFetched = time.Now()
		s.state.Loading = false
		s.state.Err = nil
	}, func(err error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.state.Err = err
		s.state.Loading = false
	})
}
