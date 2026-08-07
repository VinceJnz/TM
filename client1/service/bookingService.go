package service

import (
	"client1/v2/app/api"
	"client1/v2/app/httpProcessor"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/shopspring/decimal"
)

const bookingsApiURL = "/bookings"

type Booking struct {
	ID              int             `json:"id"`
	OwnerID         int             `json:"owner_id"`
	TripID          int             `json:"trip_id"`
	Notes           string          `json:"notes"`
	FromDate        time.Time       `json:"from_date"`
	ToDate          time.Time       `json:"to_date"`
	Participants    int             `json:"participants"` // Report generated field
	GroupBookingID  int             `json:"group_booking_id"`
	GroupBooking    string          `json:"group_booking"` // Report generated field
	BookingStatusID int             `json:"booking_status_id"`
	BookingStatus   string          `json:"booking_status"` // Report generated field
	BookingDate     time.Time       `json:"booking_date"`
	PaymentDate     time.Time       `json:"payment_date"`
	BookingPrice    decimal.Decimal `json:"booking_price"`
	TripName        string          `json:"trip_name"`    // Report generated field
	BookingCost     decimal.Decimal `json:"booking_cost"` // Report generated field
	Created         time.Time       `json:"created"`
	Modified        time.Time       `json:"modified"`
}

type BookingService struct {
	state  DataState[[]Booking]
	ttl    time.Duration
	client *httpProcessor.Client
	api    api.BookingAPI
}

func NewBookingService(api *api.Booking) *BookingService {
	return &BookingService{
		api: api,
	}
}

func (s *BookingService) GetState() DataState[[]Booking] {
	return s.state
}

func (s *BookingService) LoadIfNeeded() {
	if time.Since(s.state.LastFetched) < s.ttl {
		return
	}

	s.state.Loading = true

	go func() {
		data, err := fetchBookingsFromAPI()

		if err != nil {
			s.state.Err = err
		} else {
			s.state.Data = data
			s.state.LastFetched = time.Now()
		}

		s.state.Loading = false
	}()
}

func fetchBookingsFromAPI() ([]Booking, error) {
	// Create a channel to wait for async result
	done := make(chan struct{})

	var result []Booking
	var fetchErr error

	js.Global().Call("fetch", bookingsApiURL).
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			response := args[0]

			// Check HTTP status
			if !response.Get("ok").Bool() {
				fetchErr = fmt.Errorf("HTTP error: %d", response.Get("status").Int())
				close(done)
				return nil
			}

			// Parse JSON
			return response.Call("json")
		})).
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			data := args[0]

			// Convert JS object → Go
			jsonBytes := []byte(js.Global().Get("JSON").Call("stringify", data).String())

			err := json.Unmarshal(jsonBytes, &result)
			if err != nil {
				fetchErr = err
			}

			close(done)
			return nil
		})).
		Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
			fetchErr = fmt.Errorf("fetch failed: %v", args[0])
			close(done)
			return nil
		}))

	// Wait for async to complete
	<-done

	return result, fetchErr
}

func (s *BookingService) fetchBookingsFromAPI2() ([]Booking, error) {
	var result []Booking
	//var fetchErr error

	success := func([]Booking) {
		//return result
	}

	failure := func(err error) {
		//fetchErr = err
	}

	s.api.GetBookings(success, failure)
}
