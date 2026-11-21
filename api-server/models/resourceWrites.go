package models

import (
	"github.com/guregu/null/v5/zero"
	"github.com/shopspring/decimal"
	"time"
)

type BookingWrite struct {
	OwnerID      int                 `json:"owner_id"`
	TripID       zero.Int            `json:"trip_id"`
	Notes        string              `json:"notes"`
	FromDate     time.Time           `json:"from_date"`
	ToDate       time.Time           `json:"to_date"`
	BookingPrice decimal.NullDecimal `json:"booking_price"`
}
type TripWrite struct {
	Name            string      `json:"trip_name"`
	Location        zero.String `json:"location"`
	DifficultyID    zero.Int    `json:"difficulty_level_id"`
	FromDate        time.Time   `json:"from_date"`
	ToDate          time.Time   `json:"to_date"`
	MaxParticipants zero.Int    `json:"max_participants"`
	TripStatusID    zero.Int    `json:"trip_status_id"`
	TripTypeID      zero.Int    `json:"trip_type_id"`
	TripCostGroupID zero.Int    `json:"trip_cost_group_id"`
}

func (input TripWrite) Trip() Trip {
	return Trip{Name: input.Name, Location: input.Location, DifficultyID: input.DifficultyID, FromDate: input.FromDate, ToDate: input.ToDate, MaxParticipants: input.MaxParticipants, TripStatusID: input.TripStatusID, TripTypeID: input.TripTypeID, TripCostGroupID: input.TripCostGroupID}
}
