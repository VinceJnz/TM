package models

import (
	"database/sql"
	"time"
)

type Booking struct {
	ID              int           `json:"id" db:"id"`
	OwnerID         int           `json:"owner_id" db:"owner_id"`
	TripID          sql.NullInt32 `json:"trip_id" db:"trip_id"`
	Notes           string        `json:"notes" db:"notes"`
	FromDate        time.Time     `json:"from_date" db:"from_date"`
	ToDate          time.Time     `json:"to_date" db:"to_date"`
	BookingStatusID int           `json:"booking_status_id" db:"booking_status_id"`
	BookingStatus   string        `json:"booking_status" db:"status"`
	Created         time.Time     `json:"created" db:"created"`
	Modified        time.Time     `json:"modified" db:"modified"`
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
