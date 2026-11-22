package helpers

import (
	"api-server/v2/models"
	"encoding/json"
	"errors"
	"net/http"
)

// Booking writes cannot accept payment status, payment dates or ledger values.
// Price overrides are an administrative capability, never a member-supplied quote.
func DecodeBookingWrite(w http.ResponseWriter, r *http.Request, anyScope bool, creating bool) (models.Booking, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input struct {
		models.BookingWrite
		Price json.RawMessage `json:"booking_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return models.Booking{}, err
	}
	record := models.Booking{OwnerID: input.OwnerID, TripID: input.TripID, Notes: input.Notes, FromDate: input.FromDate, ToDate: input.ToDate, BookingStatusID: int(models.Not_paid)}
	record.VoucherInput = input.VoucherCode
	if input.Price != nil {
		if err := json.Unmarshal(input.Price, &input.BookingPrice); err != nil {
			return models.Booking{}, err
		}
	}
	if anyScope && !creating {
		if input.BookingPrice.Valid && input.BookingPrice.Decimal.IsNegative() {
			return models.Booking{}, errors.New("price cannot be negative")
		}
		record.BookingPrice = input.BookingPrice
		record.PreserveBookingPrice = input.Price == nil
	}
	return record, nil
}
