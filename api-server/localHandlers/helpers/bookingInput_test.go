package helpers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemberCannotSetFinancialFields(t *testing.T) {
	for _, creating := range []bool{true, false} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"booking_status_id":3,"booking_price":0.01,"payment_date":"2026-01-01T00:00:00Z","amount_paid":100,"stripe_session_id":"forged"}`))
		b, err := DecodeBookingWrite(httptest.NewRecorder(), r, false, creating)
		if err != nil {
			t.Fatal(err)
		}
		if b.BookingStatusID != 1 || b.BookingPrice.Valid || b.PaymentDate.Valid || b.AmountPaid.Valid || b.StripeSessionID.Valid {
			t.Fatal("member supplied financial fields survived decoding")
		}
	}
}

func TestAdminPriceDoesNotSetPaidStatus(t *testing.T) {
	r := httptest.NewRequest("PUT", "/", strings.NewReader(`{"booking_price":12.34,"booking_status_id":3}`))
	b, err := DecodeBookingWrite(httptest.NewRecorder(), r, true, false)
	if err != nil || !b.BookingPrice.Valid || b.BookingPrice.Decimal.String() != "12.34" || b.BookingStatusID != 1 {
		t.Fatalf("invalid administrative input handling: %v", err)
	}
}
