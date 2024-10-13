package models

import (
	"time"

	"github.com/guregu/null/v5/zero"
)

type Trip struct {
	ID              int         `json:"id" db:"id"`
	OwnerID         int         `json:"owner_id" db:"owner_id"`
	Name            string      `json:"trip_name" db:"trip_name"`
	Location        string      `json:"location" db:"location"`
	Difficulty      zero.String `json:"difficulty_level" db:"difficulty_level"`
	FromDate        time.Time   `json:"from_date" db:"from_date"`
	ToDate          time.Time   `json:"to_date" db:"to_date"`
	MaxParticipants int         `json:"max_participants" db:"max_participants"`
	Participants    zero.Int    `json:"participants" db:"participants"`
	TripStatusID    zero.Int    `json:"trip_status_id" db:"trip_status_id"`
	TripStatus      zero.String `json:"trip_status" db:"trip_status"`
	Created         time.Time   `json:"created" db:"created"`
	Modified        time.Time   `json:"modified" db:"modified"`
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
