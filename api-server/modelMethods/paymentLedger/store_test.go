package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"sync"
	"testing"
)

const ledgerFixture = testdb.Ledger

func TestFinalizeValidatesQuoteAndSerializesReplays(t *testing.T) {
	db := testdb.Open(t, ledgerFixture)
	ctx := context.Background()
	for _, tc := range []struct {
		amount   int64
		currency string
	}{{999, "nzd"}, {1000, "usd"}} {
		if _, err := Finalize(ctx, db, "invalid", "checkout", tc.amount, tc.currency); err == nil {
			t.Fatal("invalid quote accepted")
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Finalize(ctx, db, "event-one", "checkout", 1000, "nzd"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var payments, events, status, pct int
	if err := db.Get(&payments, "SELECT count(*) FROM at_payments"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&events, "SELECT count(*) FROM at_stripe_events"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT booking_status_id,amount_paid FROM at_bookings WHERE id=11").Scan(&status, &pct); err != nil {
		t.Fatal(err)
	}
	if payments != 1 || events != 1 || status != 3 || pct != 100 {
		t.Fatal("duplicate payment or inconsistent booking state")
	}
	// A different event for the same checkout must not recreate refunded money.
	if _, err := db.Exec("INSERT INTO at_refunds(payment_id,amount) SELECT id,10 FROM at_payments"); err != nil {
		t.Fatal(err)
	}
	if _, err := Finalize(ctx, db, "event-two", "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&status, "SELECT booking_status_id FROM at_bookings WHERE id=11"); err != nil {
		t.Fatal(err)
	}
	if status != 1 {
		t.Fatal("replay resurrected refunded payment")
	}
}

func TestFinalizationFailureRollsBackAndCanRetry(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`
CREATE FUNCTION reject_payment() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic failure'; END $$;
CREATE TRIGGER fail_payment BEFORE INSERT ON at_payments FOR EACH ROW EXECUTE FUNCTION reject_payment();`)
	if _, err := Finalize(context.Background(), db, "retry-event", "checkout", 1000, "nzd"); err == nil {
		t.Fatal("failed write was accepted")
	}
	var events, status int
	if err := db.Get(&events, "SELECT count(*) FROM at_stripe_events"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&status, "SELECT booking_status_id FROM at_bookings WHERE id=11"); err != nil {
		t.Fatal(err)
	}
	if events != 0 || status != 1 {
		t.Fatal("transaction left partial state")
	}
	if _, err := db.Exec("DROP TRIGGER fail_payment ON at_payments"); err != nil {
		t.Fatal(err)
	}
	if _, err := Finalize(context.Background(), db, "retry-event", "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
}

func TestQuoteUsesExactMinorUnits(t *testing.T) {
	db := testdb.Open(t, testdb.Bookings+`UPDATE at_bookings SET booking_price=NULL,from_date='2026-01-01',to_date='2026-01-02' WHERE id=11;UPDATE at_trip_costs SET amount=0.29;`)
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	quote, err := Quote(context.Background(), tx, 11)
	if err != nil || quote != 29 {
		t.Fatalf("quote=%d, err=%v", quote, err)
	}
}

func TestManualLedgerUsesExactStatusAndRejectsExcessRefund(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`UPDATE at_bookings SET booking_price=0.29 WHERE id=11;INSERT INTO at_payments(booking_id,amount,payment_date) VALUES(11,0.29,NOW());`)
	ctx := context.Background()
	tx, err := Begin(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = Sync(ctx, tx, 11); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var status int
	if err = db.Get(&status, "SELECT booking_status_id FROM at_bookings WHERE id=11"); err != nil || status != 3 {
		t.Fatal(status, err)
	}
	tx, err = Begin(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO at_refunds(payment_id,amount) SELECT id,0.30 FROM at_payments"); err != nil {
		t.Fatal(err)
	}
	if err = Sync(ctx, tx, 11); err == nil {
		t.Fatal("excess refund accepted")
	}
	tx.Rollback()
	var refunds int
	if err = db.Get(&refunds, "SELECT COUNT(*) FROM at_refunds"); err != nil || refunds != 0 {
		t.Fatal("invalid refund persisted", err)
	}
}
