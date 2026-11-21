package service

import (
	"client1/v2/app/api"
	"client1/v2/app/httpProcessor"
	"context"
	"errors"
	"testing"
	"time"
)

type cancellableAPI struct {
	api.BookingAPI
	started chan struct{}
}

func (p *cancellableAPI) GetPage(*httpProcessor.Page, string, *[]api.BookingDTO) error {
	panic("context adapter not used")
}
func (p *cancellableAPI) GetPageContext(ctx context.Context, _ *httpProcessor.Page, _ string, _ *[]api.BookingDTO) error {
	close(p.started)
	<-ctx.Done()
	return ctx.Err()
}
func TestInvalidatingViewCancelsActivePage(t *testing.T) {
	p := &cancellableAPI{started: make(chan struct{})}
	s := NewBookingService(p)
	done := make(chan error, 1)
	go func() { _, err := s.FetchPage(&httpProcessor.Page{}, "/bookings"); done <- err }()
	<-p.started
	s.Invalidate()
	select {
	case err := <-done:
		if !errors.Is(err, ErrStale) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("view cancellation blocked")
	}
}
