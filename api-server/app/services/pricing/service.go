package pricing

import (
	store "api-server/v2/modelMethods/pricing"
	"api-server/v2/models"
	"context"
	"errors"
	"github.com/shopspring/decimal"
	"strings"
)

var ErrValidation = errors.New("invalid pricing input")

type Service struct{ Store store.Store }

func (s Service) SaveCost(ctx context.Context, id int, v models.TripCostWrite) (models.TripCost, error) {
	if id < 0 || v.TripCostGroupID <= 0 || v.MemberStatusID <= 0 || v.UserAgeGroupID <= 0 || v.SeasonID <= 0 || v.Amount.IsNegative() || !v.Amount.Equal(v.Amount.Round(2)) || v.Amount.GreaterThan(decimal.RequireFromString("99999999.99")) {
		return models.TripCost{}, ErrValidation
	}
	// Foreign keys validate references atomically in the same write.
	return s.Store.SaveCost(ctx, id, v)
}
func (s Service) SaveGroup(ctx context.Context, id int, v models.TripCostGroupWrite) (models.TripCostGroup, error) {
	v.Description = strings.TrimSpace(v.Description)
	if id < 0 || len(v.Description) == 0 || len([]rune(v.Description)) > 50 {
		return models.TripCostGroup{}, ErrValidation
	}
	return s.Store.SaveGroup(ctx, id, v)
}
