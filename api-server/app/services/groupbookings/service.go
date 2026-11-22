package groupbookings

import (
	"api-server/v2/internal/fieldrules"
	store "api-server/v2/modelMethods/groupbookings"
	"api-server/v2/models"
	"context"
	"errors"
	"strings"
)

var ErrValidation = errors.New("invalid group booking")
var ErrReferenced = errors.New("group booking is referenced")

type Service struct{ Store store.Store }

func (s Service) Save(ctx context.Context, actor, id int, v models.GroupBookingWrite) (models.GroupBooking, error) {
	v.GroupName = strings.TrimSpace(v.GroupName)
	if actor <= 0 || id < 0 || v.OwnerID < 0 || fieldrules.Validate(v) != nil {
		return models.GroupBooking{}, ErrValidation
	}
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return models.GroupBooking{}, err
	}
	defer tx.Rollback()
	if id > 0 {
		current, e := s.Store.Lock(ctx, tx, id)
		if e != nil {
			return models.GroupBooking{}, e
		}
		if v.OwnerID == 0 {
			v.OwnerID = current.OwnerID
		}
		if current.OwnerID != v.OwnerID {
			used, e := s.Store.Referenced(ctx, tx, id)
			if e != nil {
				return models.GroupBooking{}, e
			}
			if used {
				return models.GroupBooking{}, ErrReferenced
			}
		}
	} else if v.OwnerID == 0 {
		v.OwnerID = actor
	}
	result, err := s.Store.Save(ctx, tx, id, v)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s Service) Delete(ctx context.Context, actor, id int) error {
	if actor <= 0 || id <= 0 {
		return ErrValidation
	}
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = s.Store.Lock(ctx, tx, id); err != nil {
		return err
	}
	used, err := s.Store.Referenced(ctx, tx, id)
	if err != nil {
		return err
	}
	if used {
		return ErrReferenced
	}
	if err = s.Store.Delete(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
