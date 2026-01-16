package models

import (
	"time"

	"github.com/guregu/null/v5/zero"
)

type Trip struct {
	ID              int         `json:"id" db:"id"`
	OwnerID         int         `json:"owner_id" db:"owner_id"`
	Name            string      `json:"trip_name" db:"trip_name"`
	Location        zero.String `json:"location" db:"location"`
	DifficultyID    zero.Int    `json:"difficulty_level_id" db:"difficulty_level_id"`
	Difficulty      zero.String `json:"difficulty_level" db:"difficulty_level"`
	FromDate        time.Time   `json:"from_date" db:"from_date"`
	ToDate          time.Time   `json:"to_date" db:"to_date"`
	MaxParticipants zero.Int    `json:"max_participants" db:"max_participants"` // Maximum number of people allowed on the trip
	Participants    zero.Int    `json:"participants" db:"participants"`
	TripStatusID    zero.Int    `json:"trip_status_id" db:"trip_status_id"`
	TripStatus      zero.String `json:"trip_status" db:"trip_status"`
	TripTypeID      zero.Int    `json:"trip_type_id" db:"trip_type_id"`
	TripType        zero.String `json:"trip_type" db:"trip_type"`
	TripCostGroupID zero.Int    `json:"trip_cost_group_id" db:"trip_cost_group_id"`
	TripCostGroup   zero.String `json:"trip_cost_group" db:"trip_cost_group"`
	Description     zero.String `json:"description" db:"description"`
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
	ID          int       `json:"id" db:"id"`
	Level       string    `json:"level" db:"level"`
	LevelShort  string    `json:"level_short" db:"level_short"`
	Description string    `json:"description" db:"description"`
	Created     time.Time `json:"created" db:"created"`
	Modified    time.Time `json:"modified" db:"modified"`
}

type TripType struct {
	ID       int       `json:"id" db:"id"`
	Type     string    `json:"type" db:"type"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

// trip participant status list
type TripParticipantStatus struct {
	TripID          int         `json:"trip_id" db:"trip_id"`
	TripName        string      `json:"trip_name" db:"trip_name"`
	TripFrom        time.Time   `json:"from_date" db:"from_date"`
	TripTo          time.Time   `json:"to_date" db:"to_date"`
	MaxParticipants int         `json:"max_participants" db:"max_participants"`
	BookingID       zero.Int    `json:"booking_id" db:"booking_id"`
	ParticipantID   zero.Int    `json:"participant_id" db:"participant_id"`
	PersonID        zero.Int    `json:"person_id" db:"person_id"`
	PersonName      zero.String `json:"person_name" db:"person_name"`
	BookingPosition zero.Int    `json:"booking_position" db:"booking_position"`
	BookingStatus   zero.String `json:"booking_status" db:"booking_status"`
}

// TripCost represents the at_trip_costs table
type TripCost struct {
	ID              int         `db:"id" json:"id"`
	TripCostGroupID int         `db:"trip_cost_group_id" json:"trip_cost_group_id"`
	Description     zero.String `db:"description" json:"description"`
	MemberStatusID  int         `db:"member_status_id" json:"member_status_id"`
	MemberStatus    zero.String `db:"member_status" json:"member_status"`
	UserAgeGroupID  int         `db:"user_age_group_id" json:"user_age_group_id"`
	UserAgeGroup    zero.String `db:"user_age_group" json:"user_age_group"`
	SeasonID        int         `db:"season_id" json:"season_id"`
	Season          zero.String `db:"season" json:"season"`
	Amount          float64     `db:"amount" json:"amount"`
	Created         time.Time   `db:"created" json:"created"`
	Modified        time.Time   `db:"modified" json:"modified"`
}

// TripCostGroup represents the at_trip_costs table
type TripCostGroup struct {
	ID          int       `db:"id" json:"id"`
	Description string    `db:"description" json:"description"`
	Created     time.Time `db:"created" json:"created"`
	Modified    time.Time `db:"modified" json:"modified"`
}
