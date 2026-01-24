package models

import (
	"time"

	//"github.com/anz-bank/decimal"
	"github.com/guregu/null/v5/zero"
	"github.com/shopspring/decimal"
)

type BookingStatusID int

const (
	Not_paid BookingStatusID = iota
	Full_amountPaid
	Partial_amountPaid
)

type Booking struct {
	ID              int                 `json:"id" db:"id"`
	OwnerID         int                 `json:"owner_id" db:"owner_id"`
	OwnerName       zero.String         `json:"owner_username" db:"owner_username"`
	TripID          zero.Int            `json:"trip_id" db:"trip_id"`
	Notes           string              `json:"notes" db:"notes"`
	FromDate        time.Time           `json:"from_date" db:"from_date"`
	ToDate          time.Time           `json:"to_date" db:"to_date"`
	Participants    zero.Int            `json:"participants" db:"participants"`
	GroupBookingID  zero.Int            `json:"group_booking_id" db:"group_booking_id"`
	GroupBooking    zero.String         `json:"group_booking" db:"group_booking"`
	BookingStatusID int                 `json:"booking_status_id" db:"booking_status_id"`
	BookingStatus   string              `json:"booking_status" db:"status"`
	BookingDate     zero.Time           `json:"booking_date" db:"booking_date"`
	PaymentDate     zero.Time           `json:"payment_date" db:"payment_date"`
	BookingPrice    decimal.NullDecimal `json:"booking_price" db:"booking_price"`
	TripName        string              `json:"trip_name" db:"trip_name"`
	BookingCost     decimal.NullDecimal `json:"booking_cost" db:"booking_cost"` // Calculated
	StripeSessionID zero.String         `json:"stripe_session_id" db:"stripe_session_id"`
	AmountPaid      zero.Float          `json:"amount_paid" db:"amount_paid"` // Percentage of the total booking cost paid
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

type MyBooking struct {
	ID              int                 `json:"id" db:"id"`
	OwnerID         int                 `json:"owner_id" db:"owner_id"`
	Owner           zero.String         `json:"owner_name" db:"owner_name"`
	TripID          zero.Int            `json:"trip_id" db:"trip_id"`
	Notes           string              `json:"notes" db:"notes"`
	FromDate        time.Time           `json:"from_date" db:"from_date"`
	ToDate          time.Time           `json:"to_date" db:"to_date"`
	Participants    zero.Int            `json:"participants" db:"participants"`
	GroupBookingID  zero.Int            `json:"group_booking_id" db:"group_booking_id"`
	GroupBooking    zero.String         `json:"group_booking" db:"group_booking"`
	BookingStatusID int                 `json:"booking_status_id" db:"booking_status_id"`
	BookingStatus   string              `json:"booking_status" db:"status"`
	BookingDate     zero.Time           `json:"booking_date" db:"booking_date"`
	PaymentDate     zero.Time           `json:"payment_date" db:"payment_date"`
	BookingPrice    decimal.NullDecimal `json:"booking_price" db:"booking_price"`
	TripName        string              `json:"trip_name" db:"trip_name"`
	TripFromDate    time.Time           `json:"trip_from_date" db:"trip_from_date"`
	TripToDate      time.Time           `json:"trip_to_date" db:"trip_to_date"`
	BookingCost     decimal.NullDecimal `json:"booking_cost" db:"booking_cost"` // Calculated
	StripeSessionID zero.String         `json:"stripe_session_id" db:"stripe_session_id"`
	AmountPaid      zero.Float          `json:"amount_paid" db:"amount_paid"` // Percentage of the total booking cost paid
	Created         time.Time           `json:"created" db:"created"`
	Modified        time.Time           `json:"modified" db:"modified"`
}

type BookingPaymentInfo struct {
	ID                  int64       `db:"id"`
	OwnerID             int64       `db:"owner_id"`
	TripID              int64       `db:"trip_id"`
	FromDate            time.Time   `db:"from_date"`
	ToDate              time.Time   `db:"to_date"`
	BookingStatusID     zero.Int    `db:"booking_status_id"` // The status of the booking payment (Not_paid, Full_amountPaid, Partial_amount_Paid)
	BookingStatus       zero.String `json:"booking_status" db:"booking_status"`
	StripeSessionID     zero.String `db:"stripe_session_id"`
	AmountPaid          zero.Float  `db:"amount_paid"` // Percentage of the total booking cost paid
	TripName            zero.String `db:"trip_name"`
	Description         zero.String `db:"description"`
	MaxParticipants     zero.Int    `json:"max_participants" db:"max_participants"` // Maximum number of people allowed on the trip
	BookingParticipants zero.Int    `db:"participants"`                             // Number of people in the booking
	BookingCost         zero.Float  `db:"booking_cost"`                             // Total cost of the booking
	TripPersonCount     zero.Int    `db:"trip_person_count"`                        // Number of people already booked on the trip
	BookingPosition     zero.Int    `json:"booking_position" db:"booking_position"` // Position in the trip booking list
	DatePaid            zero.Time   `json:"date_paid" db:"date_paid"`
}
