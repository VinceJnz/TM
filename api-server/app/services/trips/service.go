package trips

import (
	"api-server/v2/internal/observability"
	store "api-server/v2/modelMethods/trips"
	"api-server/v2/models"
	"context"
	"errors"
	"time"
)

var ErrValidation = errors.New("invalid trip")

type Service struct{ Store store.Store }

func (s Service) Save(ctx context.Context, actor int, any bool, operation string, input models.Trip) (result models.Trip, commandErr error) {
	start := time.Now()
	defer func() {
		outcome := "success"
		if commandErr != nil {
			outcome = "failed"
		}
		observability.Event(ctx, "trip_command", map[string]interface{}{"resource": "trips", "user_id": actor, "operation": operation, "outcome": outcome, "duration_ms": time.Since(start).Milliseconds()})
	}()
	if actor <= 0 {
		return input, ErrValidation
	}
	if operation != "delete" && (input.Name == "" || input.FromDate.IsZero() || input.ToDate.IsZero() || input.ToDate.Before(input.FromDate) || input.MaxParticipants.Int64 < 0) {
		return input, ErrValidation
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return input, err
	}
	defer tx.Rollback()
	if operation != "create" {
		if err = s.Store.Lock(ctx, tx, input.ID, actor, any); err != nil {
			return input, err
		}
	}
	switch operation {
	case "create":
		input.ID, err = s.Store.Create(ctx, tx, actor, input)
		input.OwnerID = actor
	case "update":
		current, e := s.Store.Version(ctx, tx, input.ID)
		if e != nil {
			return input, e
		}
		if input.Version <= 0 || input.Version != current {
			return input, models.ErrStaleRecord
		}
		var fit bool
		fit, err = s.Store.BookingsFit(ctx, tx, input)
		if err == nil && !fit {
			return input, ErrValidation
		}
		if err == nil {
			err = s.Store.Update(ctx, tx, actor, any, &input)
		}
	case "delete":
		err = s.Store.Delete(ctx, tx, actor, any, input.ID)
	default:
		return input, ErrValidation
	}
	if err != nil {
		return input, err
	}
	if operation != "delete" {
		input.Version, err = s.Store.Version(ctx, tx, input.ID)
		if err != nil {
			return input, err
		}
	}
	return input, tx.Commit()
}
