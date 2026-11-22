package vouchers

import (
	store "api-server/v2/modelMethods/vouchers"
	"api-server/v2/models"
	"context"
	"errors"

	"math"
	"strings"
)

var ErrValidation = errors.New("invalid voucher")

type Service struct{ Store store.Store }

func (s Service) Save(ctx context.Context, id int, v models.VoucherWrite) (models.BookingVoucher, error) {
	v.Code = strings.ToUpper(strings.TrimSpace(v.Code))
	result := models.BookingVoucher{}
	if id < 0 || v.Code == "" || len(v.Code) > 64 || (v.DiscountPercent == nil) == (v.FixedCost == nil) {
		return result, ErrValidation
	}
	if p := v.DiscountPercent; p != nil {
		if math.IsNaN(*p) || math.IsInf(*p, 0) || *p < 0 || *p > 100 {
			return result, ErrValidation
		}
	}
	if p := v.FixedCost; p != nil && p.Validate() != nil {
		return result, ErrValidation
	}
	return s.Store.Save(ctx, id, v)
}
