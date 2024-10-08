package models

import "time"

type Trip struct {
	ID              int       `json:"id" db:"id"`
	OwnerID         int       `json:"owner_id" db:"owner_id"`
	Name            string    `json:"trip_name" db:"trip_name"`
	Location        string    `json:"location" db:"location"`
	Difficulty      string    `json:"trip_difficulty" db:"trip_difficulty"`
	FromDate        time.Time `json:"from_date" db:"from_date"`
	ToDate          time.Time `json:"to_date" db:"to_date"`
	MaxParticipants int       `json:"max_participants" db:"max_participants"`
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
