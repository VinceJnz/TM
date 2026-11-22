package models

import "time"

type VoucherWrite struct {
	Code            string     `json:"code"`
	DiscountPercent *float64   `json:"discount_percent"`
	FixedCost       *Money     `json:"fixed_cost"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	IsActive        bool       `json:"is_active"`
}
