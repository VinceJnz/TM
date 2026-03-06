package viewHelpers

import "github.com/shopspring/decimal"

// CalculateVoucherBookingPrice applies fixed-cost or discount voucher values to a base booking price.
// Values are clamped to non-negative and rounded to 2 decimal places.
func CalculateVoucherBookingPrice(basePrice decimal.Decimal, discountPercent, fixedCost *float64) decimal.Decimal {
	if fixedCost != nil {
		price := decimal.NewFromFloat(*fixedCost)
		if price.IsNegative() {
			price = decimal.Zero
		}
		return price.RoundBank(2)
	}

	discount := decimal.Zero
	if discountPercent != nil {
		discount = decimal.NewFromFloat(*discountPercent)
	}

	multiplier := decimal.NewFromInt(1).Sub(discount.Div(decimal.NewFromInt(100)))
	if multiplier.IsNegative() {
		multiplier = decimal.Zero
	}

	return basePrice.Mul(multiplier).RoundBank(2)
}
