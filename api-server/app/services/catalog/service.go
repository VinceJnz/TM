package catalog

import (
	store "api-server/v2/modelMethods/catalog"
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrValidation = errors.New("invalid catalog value")

type Service[T any] struct{ Store store.Store[T] }

func (s Service[T]) Save(ctx context.Context, id int, v T) (T, error) {
	if id < 0 {
		return v, ErrValidation
	}
	args := s.Store.Definition.CreateArgs(&v)
	for i, arg := range args {
		if wrapped, ok := arg.(driver.Valuer); ok {
			value, err := wrapped.Value()
			if err != nil {
				return v, ErrValidation
			}
			arg = value
		}
		if text, ok := arg.(string); ok {
			if i == 0 && strings.TrimSpace(text) == "" {
				return v, ErrValidation
			}
			if i < len(s.Store.Definition.Limits) && s.Store.Definition.Limits[i] > 0 && utf8.RuneCountInString(text) > s.Store.Definition.Limits[i] {
				return v, ErrValidation
			}
		}
	}
	return s.Store.Save(ctx, id, v)
}
