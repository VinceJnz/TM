package accounts

import (
	profiles "api-server/v2/app/services/profiles"
	"api-server/v2/internal/fieldrules"
	store "api-server/v2/modelMethods/accounts"
	"api-server/v2/models"
	"context"

	"errors"

	"strings"
	"time"
)

var ErrValidation = errors.New("invalid account values")
var ErrReason = errors.New("sensitive account changes require a reason")

type Service struct{ Store store.Store }

func (s Service) Save(ctx context.Context, actor, id int, v models.AccountWrite, reason string) (models.AccountDTO, error) {
	v.Name = strings.TrimSpace(v.Name)
	v.Username = strings.TrimSpace(v.Username)
	v.MemberCode = strings.TrimSpace(v.MemberCode)
	reason = strings.TrimSpace(reason)
	email, err := profiles.CanonicalEmail(v.Email)
	if actor <= 0 || id < 0 || err != nil || fieldrules.Validate(v) != nil || v.UserAgeGroupID < 0 || v.MemberStatusID < 0 || v.AccountStatusID < 0 || len(reason) > 500 || (!v.BirthDate.IsZero() && (v.BirthDate.Year() < 1900 || v.BirthDate.After(time.Now()))) {
		return models.AccountDTO{}, ErrValidation
	}
	v.Email = email
	tx, err := s.Store.Begin(ctx, actor, reason)
	if err != nil {
		return models.AccountDTO{}, err
	}
	defer tx.Rollback()
	sensitive := false
	if id > 0 {
		current, e := s.Store.Lock(ctx, tx, id)
		if e != nil {
			return models.AccountDTO{}, e
		}
		if v.Version <= 0 || current.Version != v.Version {
			return models.AccountDTO{}, models.ErrStaleRecord
		}
		sensitive = !strings.EqualFold(current.Email, email) || current.Username != v.Username || current.AccountStatusID != v.AccountStatusID
		if sensitive && reason == "" {
			return models.AccountDTO{}, ErrReason
		}
	}
	result, err := s.Store.Save(ctx, tx, id, v)
	if err != nil {
		return result, err
	}
	if sensitive {
		if err = s.Store.Revoke(ctx, tx, id); err != nil {
			return result, err
		}
	}
	return result, tx.Commit()
}
func (s Service) Delete(ctx context.Context, actor, id int, reason string) error {
	reason = strings.TrimSpace(reason)
	if actor <= 0 || id <= 0 || len(reason) > 500 {
		return ErrValidation
	}
	if reason == "" {
		return ErrReason
	}
	tx, err := s.Store.Begin(ctx, actor, reason)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = s.Store.Lock(ctx, tx, id); err != nil {
		return err
	}
	if err = s.Store.Delete(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
