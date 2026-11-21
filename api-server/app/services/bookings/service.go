package bookings

import (
	"api-server/v2/internal/observability"
	store "api-server/v2/modelMethods/bookings"
	"api-server/v2/modelMethods/notifications"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrValidation = errors.New("invalid booking")

type Mailer interface {
	SendMail(string, string, string) (bool, error)
}
type Service struct {
	Store      store.Store
	Email      Mailer
	ActorEmail string
}

func (s Service) Save(ctx context.Context, actor int, any bool, operation string, input models.Booking) (result models.Booking, commandErr error) {
	start := time.Now()
	defer func() {
		outcome := "success"
		if commandErr != nil {
			outcome = "failed"
		}
		observability.Event(ctx, "booking_command", map[string]interface{}{"booking_id": result.ID, "user_id": actor, "operation": operation, "outcome": outcome, "duration_ms": time.Since(start).Milliseconds()})
	}()
	if actor <= 0 {
		return input, ErrValidation
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return input, err
	}
	defer tx.Rollback()
	var bounds store.Bounds
	if operation == "create" {
		bounds, err = s.Store.LockTrip(ctx, tx, int(input.TripID.Int64))
		input.OwnerID = actor
	} else {
		target := 0
		if any && operation == "update" {
			target = int(input.TripID.Int64)
		}
		var current models.Booking
		current, bounds, err = s.Store.Lock(ctx, tx, input.ID, target)
		if err == nil && !any && current.OwnerID != actor {
			return input, sql.ErrNoRows
		}
		if !any {
			input.OwnerID = current.OwnerID
			input.TripID = current.TripID
			input.BookingPrice = current.BookingPrice
		}
	}
	if err != nil {
		return input, err
	}
	if operation != "delete" && (input.FromDate.IsZero() || input.ToDate.IsZero() || input.ToDate.Before(input.FromDate) || input.FromDate.Before(bounds.From) || input.ToDate.After(bounds.To) || (input.BookingPrice.Valid && input.BookingPrice.Decimal.IsNegative())) {
		return input, ErrValidation
	}
	switch operation {
	case "create":
		input.ID, err = s.Store.Create(ctx, tx, actor, input)
	case "update":
		err = s.Store.Update(ctx, tx, actor, any, input)
	case "delete":
		err = s.Store.Delete(ctx, tx, actor, any, input.ID)
	default:
		return input, ErrValidation
	}
	if err != nil {
		return input, err
	}
	if operation == "create" && s.ActorEmail != "" {
		if err = notifications.EnqueueBooking(ctx, tx, input.ID, int(input.TripID.Int64), s.ActorEmail); err != nil {
			return input, err
		}
	}
	if err = tx.Commit(); err != nil {
		return input, err
	}

	return input, nil
}
