package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guregu/null/v5/zero"
)

func TestManualVoucherRedemptionFreezesAgreedTotal(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`
		UPDATE at_bookings SET booking_price=100, voucher_code='SAVE' WHERE id=22;
		INSERT INTO at_booking_vouchers(code,discount_percent,fixed_cost,expiry_date,is_active) VALUES('SAVE',20,NULL,'2030-07-01',true);
	`)
	ctx := context.Background()
	input := PaymentWrite{BookingID: 22, Amount: amount("40"), PaymentDate: zero.TimeFrom(time.Date(2030, 7, 1, 0, 0, 0, 0, time.UTC))}
	if _, err := WritePayment(ctx, db, "create", 0, input); !errors.Is(err, ErrVoucherUnavailable) {
		t.Fatalf("payment on expiry date: %v", err)
	}
	input.PaymentDate = zero.TimeFrom(input.PaymentDate.Time.AddDate(0, 0, -1))
	if _, err := WritePayment(ctx, db, "create", 0, input); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE at_booking_vouchers SET expiry_date='2030-06-30',discount_percent=5 WHERE code='SAVE'`); err != nil {
		t.Fatal(err)
	}
	input.PaymentDate = zero.TimeFrom(input.PaymentDate.Time.AddDate(0, 0, 2))
	if _, err := WritePayment(ctx, db, "create", 0, input); err != nil {
		t.Fatal(err)
	}
	var paid bool
	err := db.Get(&paid, `SELECT b.booking_status_id=3 AND b.amount_paid=100 AND b.booking_price=100 AND r.amount_minor=8000
		AND r.source='manual' AND (r.used_at AT TIME ZONE 'Pacific/Auckland')::date=DATE '2030-06-30'
		FROM at_bookings b JOIN at_voucher_redemptions r ON r.booking_id=b.id WHERE b.id=22`)
	if err != nil || !paid {
		t.Fatalf("discounted payment did not settle against a frozen total: %v", err)
	}
}

func TestStripeVoucherUsesVerifiedPaymentTimeAndIsRetrySafe(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`
		UPDATE at_bookings SET voucher_code='SAVE' WHERE id=11;
		INSERT INTO at_checkout_attempts(id,booking_id,session_id,state,voucher_id,voucher_cutoff)
		VALUES('attempt',11,'checkout','attached',42,'2030-07-01 00:00+12');
	`)
	ctx := context.Background()
	cutoff := time.Date(2030, 6, 30, 12, 0, 0, 0, time.UTC)
	if _, err := Finalize(ctx, db, "late", "checkout", 1000, "nzd", cutoff); !errors.Is(err, ErrVoucherUnavailable) {
		t.Fatalf("payment at cutoff: %v", err)
	}
	for _, event := range []string{"paid", "paid", "retry"} {
		if _, err := Finalize(ctx, db, event, "checkout", 1000, "nzd", cutoff.Add(-time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM at_voucher_redemptions WHERE booking_id=11 AND source='stripe' AND reference='checkout' AND amount_minor=1000`)
	if err != nil || count != 1 {
		t.Fatalf("expected one redemption, got %d: %v", count, err)
	}
}
