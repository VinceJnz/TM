package service

import (
	"client1/v2/app/api"
	"client1/v2/app/httpProcessor"
	"errors"
	"testing"
	"time"
)

type pagedAPI struct {
	api.BookingAPI
	started chan struct{}
	release chan struct{}
}

func (p *pagedAPI) GetPage(page *httpProcessor.Page, path string, rows *[]api.BookingDTO) error {
	p.started <- struct{}{}
	<-p.release
	page.Offset = 9
	*rows = []api.BookingDTO{{ID: 99}}
	return nil
}
func TestPageInvalidationDiscardsLateRowsAndCursor(t *testing.T) {
	p := &pagedAPI{started: make(chan struct{}, 1), release: make(chan struct{})}
	s := NewBookingService(p)
	page := httpProcessor.Page{}
	done := make(chan error, 1)
	go func() { _, err := s.FetchPage(&page, "/trips/1/bookings"); done <- err }()
	<-p.started
	s.Invalidate()
	close(p.release)
	select {
	case err := <-done:
		if !errors.Is(err, ErrStale) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked page")
	}
	if page.Offset != 0 || len(s.GetState().Data) != 0 || s.GetState().Loading {
		t.Fatal("stale state committed")
	}
}
