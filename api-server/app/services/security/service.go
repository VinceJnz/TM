package security

import (
	"api-server/v2/app/authorization"
	"api-server/v2/internal/fieldrules"
	store "api-server/v2/modelMethods/security"
	"api-server/v2/models"
	"context"
	"errors"
	"strings"
)

var ErrValidation = errors.New("invalid security change")

type Service struct{ Store store.Store }

func (s Service) SaveGrant(ctx context.Context, actor, id int, v models.GroupResourceWrite) (models.GroupResource, error) {
	if actor <= 0 || id < 0 || fieldrules.Validate(v) != nil {
		return models.GroupResource{}, ErrValidation
	}
	tx, err := s.Store.Begin(ctx, actor)
	if err != nil {
		return models.GroupResource{}, err
	}
	defer tx.Rollback()
	resource, method, scope, err := s.Store.GrantShape(ctx, tx, v)
	if err != nil {
		return models.GroupResource{}, err
	}
	method = strings.ToUpper(method)
	if (method != "GET" && method != "POST" && method != "PUT" && method != "DELETE") || !authorization.SupportsScope(resource, method, scope) {
		return models.GroupResource{}, ErrValidation
	}
	result, err := s.Store.SaveGrant(ctx, tx, id, v)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s Service) SaveMember(ctx context.Context, actor, id int, v models.UserGroupWrite) (models.UserGroup, error) {
	if actor <= 0 || id < 0 || fieldrules.Validate(v) != nil {
		return models.UserGroup{}, ErrValidation
	}
	tx, err := s.Store.Begin(ctx, actor)
	if err != nil {
		return models.UserGroup{}, err
	}
	defer tx.Rollback()
	result, err := s.Store.SaveMember(ctx, tx, id, v)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s Service) Delete(ctx context.Context, actor, id int, grant bool) error {
	if actor <= 0 || id <= 0 {
		return ErrValidation
	}
	tx, err := s.Store.Begin(ctx, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if grant {
		err = s.Store.DeleteGrant(ctx, tx, id)
	} else {
		err = s.Store.DeleteMember(ctx, tx, id)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
