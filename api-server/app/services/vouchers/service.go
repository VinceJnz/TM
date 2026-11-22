package vouchers

import (
	store "api-server/v2/modelMethods/vouchers"
	"api-server/v2/models"
	"context"
	"errors"
	"github.com/shopspring/decimal"
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
	if p := v.FixedCost; p != nil {
		if math.IsNaN(*p) || math.IsInf(*p, 0) {
			return result, ErrValidation
		}
		if (models.Money{Decimal: decimal.NewFromFloat(*p)}).Validate() != nil {
			return result, ErrValidation
		}
	}
	result = models.BookingVoucher{Code: v.Code, DiscountPercent: v.DiscountPercent, FixedCost: v.FixedCost, ExpiryDate: v.ExpiryDate, IsActive: v.IsActive}
	return s.Store.Save(ctx, id, result)
}
