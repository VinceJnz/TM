package authentication

import (
	"api-server/v2/models"
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrAgeGroupUnavailable = errors.New("invalid age group")
	ErrAgeGroupMismatch    = errors.New("birthdate does not match the selected age group")
)

func (s Service) ValidateAgeGroup(ctx context.Context, birth time.Time, id int64) error {
	label, err := s.Store.AgeGroup(ctx, nil, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAgeGroupUnavailable, err)
	}
	if !models.AgeGroupMatches(birth, label, time.Now()) {
		return ErrAgeGroupMismatch
	}
	return nil
}
