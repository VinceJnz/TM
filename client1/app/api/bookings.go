package api

import (
	"client1/v2/app/httpProcessor"
	"client1/v2/service"
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
	GetBookings(onSuccess func([]service.Booking), onError func(error))
	CreateBooking(req BookingRequest, onSuccess func(service.Booking), onError func(error))
	DeleteBooking(id int, onSuccess func(), onError func(error))
	UpdateBooking(id int, req BookingRequest, onSuccess func(service.Booking), onError func(error))
}

type Booking struct {
	client *httpProcessor.Client
	ApiUrl string
}

func (a *Booking) GetBookings(onSuccess func([]service.Booking), onError func(error)) {
	var records []service.Booking
	callbacks := httpProcessor.Callbacks{
		OnSuccess: func() { onSuccess(records) },
		OnError:   onError,
	}

	//if a.ParentData.ID != 0 { // This creates a URL that gets items for a specific parent record.
	//	localApiURL = "/trips/" + strconv.Itoa(a.ParentData.ID) + a.ApiUrl
	//}
	a.client.NewRequest2(http.MethodGet, a.ApiUrl, &records, nil, callbacks)
}

func (a *Booking) CreateBooking() {}
func (a *Booking) DeleteBooking() {}
func (a *Booking) UpdateBooking(id int, item BookingRequest, onSuccess func(), onError func(error)) {
	callbacks := httpProcessor.Callbacks{
		OnSuccess: onSuccess,
		OnError:   onError,
	}
	a.client.NewRequest2(http.MethodPut, a.ApiUrl+"/"+strconv.Itoa(item.ID), nil, &item, callbacks)
}
