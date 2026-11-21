package pricing

import (
	"api-server/v2/internal/fieldrules"
	store "api-server/v2/modelMethods/pricing"
	"api-server/v2/models"
	"context"
	"errors"
	"strings"
)

var ErrValidation = errors.New("invalid pricing input")

type Service struct{ Store store.Store }

func (s Service) SaveCost(ctx context.Context, id int, v models.TripCostWrite) (models.TripCost, error) {
	if id < 0 || fieldrules.Validate(v) != nil || v.Amount.Validate() != nil {
		return models.TripCost{}, ErrValidation
	}
	// Foreign keys validate references atomically in the same write.
	return s.Store.SaveCost(ctx, id, v)
}
func (s Service) SaveGroup(ctx context.Context, id int, v models.TripCostGroupWrite) (models.TripCostGroup, error) {
	v.Description = strings.TrimSpace(v.Description)
	if id < 0 || fieldrules.Validate(v) != nil {
		return models.TripCostGroup{}, ErrValidation
	}
	return s.Store.SaveGroup(ctx, id, v)
}
