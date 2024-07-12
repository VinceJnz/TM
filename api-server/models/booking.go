package models

import "time"

type Booking struct {
	ID              int       `json:"id" db:"ID"`
	OwnerID         int       `json:"owner_id" db:"Owner_ID"`
	Notes           string    `json:"notes" db:"Notes"`
	FromDate        time.Time `json:"from_date" db:"From_date"`
	ToDate          time.Time `json:"to_date" db:"To_date"`
	BookingStatusID int       `json:"booking_status_id" db:"Booking_status_ID"`
	Created         time.Time `json:"created" db:"Created"`
	Modified        time.Time `json:"modified" db:"Modified"`
}

type BookingStatus struct {
	ID     int    `json:"id" db:"ID"`
	Status string `json:"status" db:"status"`
}

type BookingUser struct {
	ID        int       `json:"id" db:"ID"`
	OwnerID   int       `json:"owner_id" db:"Owner_ID"`
	BookingID int       `json:"booking_id" db:"Booking_ID"`
	UserID    int       `json:"user_id" db:"User_ID"`
	Notes     string    `json:"notes" db:"Notes"`
	Created   time.Time `json:"created" db:"Created"`
	Modified  time.Time `json:"modified" db:"Modified"`
}
