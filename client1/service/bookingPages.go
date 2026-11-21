package service

import (
	"client1/v2/app/api"
	"client1/v2/app/httpProcessor"
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrStale = errors.New("booking request superseded")

type pageAPI interface {
	GetPage(*httpProcessor.Page, string, *[]api.BookingDTO) error
}

type contextPageAPI interface {
	GetPageContext(context.Context, *httpProcessor.Page, string, *[]api.BookingDTO) error
}

// FetchPage owns load/error state. Transport writes to a private page copy so
// superseded responses cannot move the visible pagination cursor.
func (s *BookingService) FetchPage(page *httpProcessor.Page, path string) ([]Booking, error) {
	adapter, ok := s.api.(pageAPI)
	if !ok {
		return nil, fmt.Errorf("booking page adapter unavailable")
	}
	s.mu.Lock()
	key := fmt.Sprintf("%s:%d:%d", path, page.Offset, page.Limit)
	if s.pageKey == key && !s.state.LastFetched.IsZero() && time.Since(s.state.LastFetched) < s.ttl {
		rows := append([]Booking(nil), s.state.Data...)
		s.mu.Unlock()
		return rows, nil
	}
	s.generation++
	if s.pageCancel != nil {
		s.pageCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.pageCancel = cancel
	defer cancel()
	generation := s.generation
	pending := page.Snapshot()
	s.state.Loading = true
	s.state.Err = nil
	s.mu.Unlock()
	var rows []Booking
	var err error
	if contextual, ok := s.api.(contextPageAPI); ok {
		err = contextual.GetPageContext(ctx, &pending, path, &rows)
	} else {
		err = adapter.GetPage(&pending, path, &rows)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if generation != s.generation {
		return nil, ErrStale
	}
	s.state.Loading = false
	s.state.Err = err
	if err != nil {
		return nil, err
	}
	*page = pending
	s.pageKey = key
	s.state.Data = append([]Booking(nil), rows...)
	s.state.LastFetched = time.Now()
	return append([]Booking(nil), rows...), nil
}
