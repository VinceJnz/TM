package models

import "time"

type Trip struct {
	ID              string    `json:"id" db:"trip_id"`
	OwnerID         int       `json:"owner_id" db:"owner_id"`
	Name            string    `json:"trip_name" db:"trip_name"`
	Location        string    `json:"location" db:"location"`
	DifficultyID    int       `json:"trip_difficulty_id" db:"trip_difficulty_id"`
	Difficulty      string    `json:"trip_difficulty" db:"trip_difficulty"`
	StartDate       time.Time `json:"start_date" db:"start_date"`
	EndDate         time.Time `json:"end_date" db:"end_date"`
	MaxParticipants int       `json:"max_participants" db:"max_participants"`
	TripStatusID    int       `json:"trip_status_id" db:"trip_status_id"`
	TripStatus      string    `json:"trip_status" db:"trip_status"`
	Created         time.Time `json:"created" db:"created"`
	Modified        time.Time `json:"modified" db:"modified"`
}

type TripStatus struct {
	ID       int       `json:"id" db:"id"`
	Status   string    `json:"status" db:"status"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type TripDificulty struct {
	ID       int       `json:"id" db:"id"`
	Level    string    `json:"level" db:"level"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type TripBooking struct {
	ID          int       `json:"id" db:"id"`
	OwnerID     int       `json:"owner_id" db:"owner_id"`
	TripID      int       `json:"trip_id" db:"trip_id"`
	BookingID   int       `json:"booking_id" db:"booking_id"`
	BookingDesc string    `json:"booking_desc" db:"booking_desc"`
	Created     time.Time `json:"created" db:"created"`
	Modified    time.Time `json:"modified" db:"modified"`
}
