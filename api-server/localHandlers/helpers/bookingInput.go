package helpers

import (
	"api-server/v2/models"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/guregu/null/v5/zero"
	"github.com/shopspring/decimal"
)

// Booking writes cannot accept payment status, payment dates or ledger values.
// Price overrides are an administrative capability, never a member-supplied quote.
func DecodeBookingWrite(w http.ResponseWriter, r *http.Request, anyScope bool, creating bool) (models.Booking, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input struct {
		OwnerID  int                 `json:"owner_id"`
		TripID   zero.Int            `json:"trip_id"`
		Notes    string              `json:"notes"`
		FromDate time.Time           `json:"from_date"`
		ToDate   time.Time           `json:"to_date"`
		Price    decimal.NullDecimal `json:"booking_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return models.Booking{}, err
	}
	record := models.Booking{OwnerID: input.OwnerID, TripID: input.TripID, Notes: input.Notes, FromDate: input.FromDate, ToDate: input.ToDate, BookingStatusID: int(models.Not_paid)}
	if anyScope && !creating {
		if input.Price.Valid && input.Price.Decimal.IsNegative() {
			return models.Booking{}, errors.New("price cannot be negative")
		}
		record.BookingPrice = input.Price
	}
	return record, nil
}
