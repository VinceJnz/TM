package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"context"
	"errors"
	"github.com/shopspring/decimal"
	"sync"
	"sync/atomic"
	"testing"
)

func amount(value string) models.Money {
	return models.Money{Decimal: decimal.RequireFromString(value)}
}

func TestProviderPaymentsCannotBeRewrittenOrDeleted(t *testing.T) {
	db := testdb.Open(t, ledgerFixture)
	ctx := context.Background()
	if _, err := Finalize(ctx, db, "immutable-event", "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
	var id int
	if err := db.Get(&id, `SELECT id FROM at_payments WHERE stripe_session_id='checkout'`); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"update", "delete"} {
		if _, err := WritePayment(ctx, db, operation, id, PaymentWrite{BookingID: 22, Amount: amount("1")}); !errors.Is(err, ErrLedgerConflict) {
			t.Fatalf("%s: %v", operation, err)
		}
	}
	var bookingID int
	var value string
	if err := db.QueryRow(`SELECT booking_id,amount::text FROM at_payments WHERE id=$1`, id).Scan(&bookingID, &value); err != nil {
		t.Fatal(err)
	}
	if bookingID != 11 || !decimal.RequireFromString(value).Equal(decimal.NewFromInt(10)) {
		t.Fatal("provider payment changed")
	}
	if _, err := Finalize(ctx, db, "immutable-replay", "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
}
func TestFinancialCommandsRejectReassignment(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`ALTER TABLE at_refunds ADD COLUMN refund_date timestamptz,ADD COLUMN reason text,ADD COLUMN external_ref text;`)
	ctx := context.Background()
	payment, err := WritePayment(ctx, db, "create", 0, PaymentWrite{BookingID: 11, Amount: amount("10")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = WriteRefund(ctx, db, "create", 0, RefundWrite{PaymentID: payment, Amount: amount("11")}); !errors.Is(err, ErrLedgerConflict) {
		t.Fatal(err)
	}
	var count int
	if err = db.Get(&count, "SELECT count(*) FROM at_refunds"); err != nil || count != 0 {
		t.Fatal("failed refund persisted", err)
	}
	refund, err := WriteRefund(ctx, db, "create", 0, RefundWrite{PaymentID: payment, Amount: amount("2")})
	if err != nil {
		t.Fatal(err)
	}
	otherPayment, err := WritePayment(ctx, db, "create", 0, PaymentWrite{BookingID: 22, Amount: amount("0")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = WriteRefund(ctx, db, "update", refund, RefundWrite{PaymentID: otherPayment, Amount: amount("1")}); !errors.Is(err, ErrLedgerConflict) {
		t.Fatalf("refund reassignment: %v", err)
	}
	if _, err = WriteRefund(ctx, db, "update", refund, RefundWrite{PaymentID: payment, Amount: amount("3")}); err != nil {
		t.Fatal(err)
	}
	if _, err = WriteRefund(ctx, db, "delete", refund, RefundWrite{}); err != nil {
		t.Fatal(err)
	}
	if _, err = WritePayment(ctx, db, "update", payment, PaymentWrite{BookingID: 22, Amount: amount("10")}); !errors.Is(err, ErrLedgerConflict) {
		t.Fatalf("payment reassignment: %v", err)
	}
	if _, err = WritePayment(ctx, db, "update", payment, PaymentWrite{BookingID: 11, Amount: amount("10")}); err != nil {
		t.Fatal(err)
	}
	var oldStatus, newStatus int
	db.Get(&oldStatus, "SELECT booking_status_id FROM at_bookings WHERE id=11")
	db.Get(&newStatus, "SELECT booking_status_id FROM at_bookings WHERE id=22")
	if oldStatus != 3 || newStatus != 1 {
		t.Fatal("reassignment changed booking states", oldStatus, newStatus)
	}
}
func TestConcurrentRefundCommandsCannotExceedBalance(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`ALTER TABLE at_refunds ADD COLUMN refund_date timestamptz,ADD COLUMN reason text,ADD COLUMN external_ref text;`)
	ctx := context.Background()
	payment, err := WritePayment(ctx, db, "create", 0, PaymentWrite{BookingID: 11, Amount: amount("10")})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var accepted atomic.Int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := WriteRefund(ctx, db, "create", 0, RefundWrite{PaymentID: payment, Amount: amount("6")})
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrLedgerConflict) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatal("concurrent refunds exceeded payment")
	}
}
