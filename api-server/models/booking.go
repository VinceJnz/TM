package models

import (
	"time"

	//"github.com/anz-bank/decimal"
	"github.com/guregu/null/v5/zero"
	"github.com/shopspring/decimal"
)

type Booking struct {
	ID              int                 `json:"id" db:"id"`
	OwnerID         int                 `json:"owner_id" db:"owner_id"`
	TripID          zero.Int            `json:"trip_id" db:"trip_id"`
	Notes           string              `json:"notes" db:"notes"`
	FromDate        time.Time           `json:"from_date" db:"from_date"`
	ToDate          time.Time           `json:"to_date" db:"to_date"`
	Participants    zero.Int            `json:"participants" db:"participants"` // Calculated
	GroupBookingID  zero.Int            `json:"group_booking_id" db:"group_booking_id"`
	GroupBooking    zero.String         `json:"group_booking" db:"group_booking"` // Calculated
	BookingStatusID int                 `json:"booking_status_id" db:"booking_status_id"`
	BookingStatus   string              `json:"booking_status" db:"status"` // Calculated
	BookingDate     zero.Time           `json:"booking_date" db:"booking_date"`
	PaymentDate     zero.Time           `json:"payment_date" db:"payment_date"`
	BookingPrice    decimal.NullDecimal `json:"booking_price" db:"booking_price"`
	TripName        string              `json:"trip_name" db:"trip_name"`       // Calculated
	BookingCost     decimal.NullDecimal `json:"booking_cost" db:"booking_cost"` // Calculated
	Created         time.Time           `json:"created" db:"created"`
	Modified        time.Time           `json:"modified" db:"modified"`
}

type BookingStatus struct {
	ID       int       `json:"id" db:"id"`
	Status   string    `json:"status" db:"status"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type BookingPeople struct {
	ID        int       `json:"id" db:"id"`
	OwnerID   int       `json:"owner_id" db:"owner_id"`
	BookingID int       `json:"booking_id" db:"booking_id"`
	PersonID  int       `json:"person_id" db:"person_id"`
	Person    string    `json:"person_name" db:"person_name"`
	Notes     string    `json:"notes" db:"notes"`
	Created   time.Time `json:"created" db:"created"`
	Modified  time.Time `json:"modified" db:"modified"`
}

type GroupBooking struct {
	ID        int    `db:"id" json:"id"`
	GroupName string `db:"group_name" json:"group_name"`
	OwnerID   int    `db:"owner_id" json:"owner_id"`
	Created   string `db:"created" json:"created"`
	Modified  string `db:"modified" json:"modified"`
}
