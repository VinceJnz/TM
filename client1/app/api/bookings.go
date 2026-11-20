package api

import (
	"client1/v2/app/httpProcessor"

	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

type BookingRequest struct {
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

type BookingDTO struct {
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

type BookingAPI interface {
	GetBookings(onSuccess func([]BookingDTO), onError func(error))
	CreateBooking(req BookingRequest, onSuccess func(BookingDTO), onError func(error))
	DeleteBooking(id int, onSuccess func(), onError func(error))
	UpdateBooking(id int, req BookingRequest, onSuccess func(BookingDTO), onError func(error))
}

type Booking struct {
	client *httpProcessor.Client
	ApiUrl string
}

func (a *Booking) GetBookings(onSuccess func([]BookingDTO), onError func(error)) {
	var records []BookingDTO
	callbacks := httpProcessor.Callbacks{
		OnSuccess: func() { onSuccess(records) },
		OnError:   onError,
	}

	//if a.ParentData.ID != 0 { // This creates a URL that gets items for a specific parent record.
	//	localApiURL = "/trips/" + strconv.Itoa(a.ParentData.ID) + a.ApiUrl
	//}
	a.client.NewRequest2(http.MethodGet, a.ApiUrl, &records, nil, callbacks)
}

func NewBooking(client *httpProcessor.Client) *Booking {
	return &Booking{client: client, ApiUrl: "/bookings"}
}

func (a *Booking) CreateBooking(req BookingRequest, onSuccess func(BookingDTO), onError func(error)) {
	var id int
	a.client.NewRequest2(http.MethodPost, a.ApiUrl, &id, &req, httpProcessor.Callbacks{
		OnSuccess: func() { a.GetBooking(id, onSuccess, onError) }, OnError: onError})
}
func (a *Booking) GetBooking(id int, onSuccess func(BookingDTO), onError func(error)) {
	var records []BookingDTO
	a.client.NewRequest2(http.MethodGet, a.ApiUrl+"/"+strconv.Itoa(id), &records, nil, httpProcessor.Callbacks{OnSuccess: func() {
		if len(records) != 1 {
			onError(fmt.Errorf("booking %d unavailable", id))
			return
		}
		onSuccess(records[0])
	}, OnError: onError})
}
func (a *Booking) DeleteBooking(id int, onSuccess func(), onError func(error)) {
	a.client.NewRequest2(http.MethodDelete, a.ApiUrl+"/"+strconv.Itoa(id), nil, nil, httpProcessor.Callbacks{OnSuccess: onSuccess, OnError: onError})
}
func (a *Booking) UpdateBooking(id int, req BookingRequest, onSuccess func(BookingDTO), onError func(error)) {
	req.ID = id
	a.client.NewRequest2(http.MethodPut, a.ApiUrl+"/"+strconv.Itoa(id), nil, &req, httpProcessor.Callbacks{OnSuccess: func() { a.GetBooking(id, onSuccess, onError) }, OnError: onError})
}

var _ BookingAPI = (*Booking)(nil)
