package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestReviewVoucherUsesCurrentTermsAndConfirmsZeroTotal(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`
		ALTER TABLE at_trips ADD COLUMN description text;
		UPDATE at_bookings SET stripe_session_id=NULL, voucher_code='TRIP', booking_price=NULL,
			notes=NULL, from_date='2026-01-01',to_date='2026-01-02' WHERE id=11;
		INSERT INTO at_booking_vouchers(code,discount_percent,fixed_cost,expiry_date,is_active) VALUES('TRIP',10,NULL,CURRENT_DATE+30,true);
	`)
	ctx := context.Background()
	if _, err := ReviewBooking(ctx, db, 11, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("other owner could review booking: %v", err)
	}
	review, err := ReviewBooking(ctx, db, 11, 1)
	if err != nil || review.Notes != "" || review.AmountMinor != 900 || review.VoucherCode != "TRIP" || len(review.Participants) != 1 {
		t.Fatalf("unexpected review: %+v, %v", review, err)
	}
	if _, err := db.Exec(`UPDATE at_booking_vouchers SET discount_percent=20 WHERE code='TRIP'`); err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(ctx, db, 11, 1, "synthetic@example.test", "https://app.test", review.AmountMinor); !errors.Is(err, ErrQuoteChanged) {
		t.Fatalf("changed quote was accepted: %v", err)
	}
	if _, err := db.Exec(`UPDATE at_bookings SET voucher_code='TRIP' WHERE id=22`); err != nil {
		t.Fatal(err)
	}
	attempt, err := Prepare(ctx, db, 22, 2, "synthetic@example.test", "https://app.test", 800)
	if err != nil || attempt.Amount != 800 {
		t.Fatalf("current voucher terms did not reach checkout: %+v, %v", attempt, err)
	}
	if err := ConfirmFreeBooking(ctx, db, 11, 1); !errors.Is(err, ErrQuoteChanged) {
		t.Fatalf("nonzero booking was confirmed for free: %v", err)
	}
	if _, err := db.Exec(`UPDATE at_booking_vouchers SET discount_percent=100,expiry_date=CURRENT_DATE-3 WHERE code='TRIP'`); err != nil {
		t.Fatal(err)
	}
	if _, err := ReviewBooking(ctx, db, 11, 1); !errors.Is(err, ErrVoucherUnavailable) {
		t.Fatalf("expired voucher accepted: %v", err)
	}
	if held, err := ReviewBooking(ctx, db, 22, 2); err != nil || held.AmountMinor != attempt.Amount {
		t.Fatalf("checkout total changed after voucher expiry: %+v, %v", held, err)
	}
	if _, err := db.Exec(`UPDATE at_booking_vouchers SET expiry_date=CURRENT_DATE+30 WHERE code='TRIP'`); err != nil {
		t.Fatal(err)
	}
	review, err = ReviewBooking(ctx, db, 11, 1)
	if err != nil || review.AmountMinor != 0 {
		t.Fatalf("zero total quote: %+v, %v", review, err)
	}
	if err := ConfirmFreeBooking(ctx, db, 11, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("other owner could confirm booking: %v", err)
	}
	for range 2 {
		if err := ConfirmFreeBooking(ctx, db, 11, 1); err != nil {
			t.Fatal(err)
		}
	}
	var confirmed bool
	if err := db.Get(&confirmed, `SELECT booking_status_id=3 AND booking_price=0 AND free_confirmed_at IS NOT NULL
		AND EXISTS(SELECT 1 FROM at_voucher_redemptions WHERE booking_id=11 AND source='free' AND amount_minor=0)
		AND stripe_session_id IS NULL AND NOT EXISTS(SELECT 1 FROM at_payments WHERE booking_id=11)
		AND EXISTS(SELECT 1 FROM at_checkout_reservations WHERE booking_id=11 AND seats=1)
		FROM at_bookings WHERE id=11`); err != nil || !confirmed {
		t.Fatalf("zero-total booking was not confirmed atomically: %v", err)
	}
}
