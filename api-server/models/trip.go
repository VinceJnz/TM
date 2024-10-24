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
	TripTypeID      zero.Int    `json:"trip_type_id" db:"trip_type_id"`
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

type TripType struct {
	ID       int       `json:"id" db:"id"`
	Type     string    `json:"type" db:"type"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

// trip participant status list
type TripParticipantStatus struct {
	TripID        int         `json:"trip_id" db:"trip_id"`
	TripName      string      `json:"trip_name" db:"trip_name"`
	TripFrom      time.Time   `json:"from_date" db:"from_date"`
	TripTo        time.Time   `json:"to_date" db:"to_date"`
	BookingID     int         `json:"booking_id" db:"booking_id"`
	ParticipantID int         `json:"participant_id" db:"participant_id"`
	PersonID      int         `json:"person_id" db:"person_id"`
	PersonName    string      `json:"person_name" db:"person_name"`
	BookingStatus zero.String `json:"booking_status" db:"booking_status"`
}

// TripCost represents the at_trip_costs table
type TripCost struct {
	ID             int       `db:"id" json:"id"`
	TripID         int       `db:"trip_id" json:"trip_id"`
	UserAgeGroupID int       `db:"user_age_group_id" json:"user_age_group_id"`
	SeasonID       int       `db:"season_id" json:"season_id"`
	Amount         float64   `db:"amount" json:"amount"`
	Created        time.Time `db:"created" json:"created"`
	Modified       time.Time `db:"modified" json:"modified"`
}
