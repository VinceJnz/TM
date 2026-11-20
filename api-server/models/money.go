package models

import (
	"bytes"
	"errors"
	"github.com/shopspring/decimal"
)

// Money keeps the existing JSON number contract while avoiding float conversion.
type Money struct{ decimal.Decimal }

func (m Money) MarshalJSON() ([]byte, error) { return []byte(m.String()), nil }
func (m *Money) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		m.Decimal = decimal.Zero
		return nil
	}
	var value decimal.Decimal
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	if value.IsNegative() || !value.Equal(value.Round(2)) {
		return errors.New("amount must be nonnegative with at most two decimal places")
	}
	m.Decimal = value
	return nil
}
