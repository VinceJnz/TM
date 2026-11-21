package security

import (
	"api-server/v2/models"
	"context"
	"strings"
	"unicode/utf8"
)

func (s Service) SaveGroup(ctx context.Context, actor, id int, v models.GroupWrite) (models.Group, error) {
	v.Name = strings.TrimSpace(v.Name)
	v.Description = strings.TrimSpace(v.Description)
	if actor <= 0 || id < 0 || v.Name == "" || utf8.RuneCountInString(v.Name) > 45 || utf8.RuneCountInString(v.Description) > 45 {
		return models.Group{}, ErrValidation
	}
	tx, err := s.Store.Begin(ctx, actor)
	if err != nil {
		return models.Group{}, err
	}
	defer tx.Rollback()
	result, err := s.Store.SaveGroup(ctx, tx, id, v)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s Service) DeleteGroup(ctx context.Context, actor, id int) error {
	if actor <= 0 || id <= 0 {
		return ErrValidation
	}
	tx, err := s.Store.Begin(ctx, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = s.Store.DeleteGroup(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
