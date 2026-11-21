package models

import (
	"bytes"
	"errors"
	"github.com/shopspring/decimal"
)

// Money keeps the existing JSON number contract while avoiding float conversion.
type Money struct{ decimal.Decimal }

const MoneyMaximum = "99999999.99"
const MoneyScale = 2

func (m Money) Validate() error {
	if m.Exponent() < -32 || m.Exponent() > 8 {
		return errors.New("amount exponent out of range")
	}
	if m.IsNegative() || !m.Equal(m.Round(MoneyScale)) || m.GreaterThan(decimal.RequireFromString(MoneyMaximum)) {
		return errors.New("amount must be nonnegative with at most two decimal places")
	}
	return nil
}

func (m Money) MarshalJSON() ([]byte, error) { return []byte(m.String()), nil }
func (m *Money) UnmarshalJSON(data []byte) error {
	if len(data) > 128 {
		return errors.New("amount is too long")
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		m.Decimal = decimal.Zero
		return nil
	}
	var value decimal.Decimal
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	// Bound the exponent before Round/comparison can allocate huge integers.
	if err := (Money{value}).Validate(); err != nil {
		return err
	}
	m.Decimal = value
	return nil
}
