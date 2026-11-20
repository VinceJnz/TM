package service

import (
	"client1/v2/app/api"
	"errors"
	"testing"
	"time"
)

type pendingAPI struct {
	api.BookingAPI
	called  chan struct{}
	success func([]api.BookingDTO)
	failure func(error)
}

func (p *pendingAPI) GetBookings(success func([]api.BookingDTO), failure func(error)) {
	p.success = success
	p.failure = failure
	p.called <- struct{}{}
}
func awaitCall(t *testing.T, p *pendingAPI) {
	t.Helper()
	select {
	case <-p.called:
	case <-time.After(time.Second):
		t.Fatal("load did not start")
	}
}
func TestCacheDeduplicatesAndRetriesFailure(t *testing.T) {
	p := &pendingAPI{called: make(chan struct{}, 4)}
	s := NewBookingService(p)
	s.LoadIfNeeded()
	awaitCall(t, p)
	s.LoadIfNeeded()
	if len(p.called) != 0 {
		t.Fatal("duplicate pending load")
	}
	p.failure(errors.New("offline"))
	if s.GetState().Loading || s.GetState().Err == nil {
		t.Fatal("failure left blocked cache")
	}
	s.LoadIfNeeded()
	awaitCall(t, p)
	p.success([]api.BookingDTO{{ID: 1}})
	state := s.GetState()
	state.Data[0].ID = 99
	if s.GetState().Data[0].ID != 1 || s.GetState().Err != nil {
		t.Fatal("cache state leaked or stale error")
	}
	s.LoadIfNeeded()
	if s.GetState().Loading {
		t.Fatal("fresh cache reloaded")
	}
	s.Invalidate()
	s.LoadIfNeeded()
	awaitCall(t, p)
	p.success(nil)
}
