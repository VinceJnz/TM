package participants

import (
	store "api-server/v2/modelMethods/participants"
	"api-server/v2/models"
	"context"
	"errors"
	"strings"
)

var ErrValidation = errors.New("invalid participant")
var ErrLocked = errors.New("booking is locked or inaccessible")

type Service struct{ Store store.Store }

func (s Service) Mutate(ctx context.Context, actor int, scope string, id int, input models.ParticipantWrite, remove bool) (int, error) {
	if actor <= 0 || id < 0 || (scope != "own" && scope != "any") || (!remove && (input.PersonID <= 0 || len(input.Notes) > 4000)) {
		return 0, ErrValidation
	}
	// Updates derive the parent from the stored row, never the submitted booking ID.
	if id > 0 {
		existing, err := s.Store.Get(ctx, actor, scope, id)
		if err != nil {
			return 0, err
		}
		input.BookingID = existing.BookingID
	}
	if input.BookingID <= 0 {
		return 0, ErrValidation
	}
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	parent, err := s.Store.Parent(ctx, tx, input.BookingID)
	if err != nil {
		return 0, err
	}
	if parent.BookingStatusID != int(models.Not_paid) || parent.StripeSessionID.Valid || (parent.OwnerID != actor && !strings.EqualFold(scope, "any")) {
		return 0, ErrLocked
	}
	if remove {
		err = s.Store.Delete(ctx, tx, actor, scope, id)
	} else {
		id, err = s.Store.Save(ctx, tx, actor, scope, id, input)
	}
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
