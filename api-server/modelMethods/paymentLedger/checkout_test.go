package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDurableAttemptRetriesAndWebhookRecovery(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/0006_checkout_attempts.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t, ledgerFixture+`ALTER TABLE at_trips ADD COLUMN description text;UPDATE at_bookings SET stripe_session_id=NULL, from_date='2026-01-02', to_date='2026-01-04' WHERE id=11;DELETE FROM at_checkout_quotes;`+string(migration))
	lifecycle, err := os.ReadFile("../../migrations/0007_checkout_lifecycle.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(lifecycle)); err != nil {
		t.Fatal(err)
	}
	priceGuard, err := os.ReadFile("../../migrations/0009_checkout_price_guard.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(priceGuard)); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := Prepare(ctx, db, 11, 1, "one@example.test", "https://example.test/api/v1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Description != "Booking dates: 2026-01-02 to 2026-01-04" {
		t.Fatalf("expected booked dates in checkout description, got %q", first.Description)
	}
	if _, err = db.Exec("UPDATE at_bookings SET booking_price=11 WHERE id=11"); err == nil {
		t.Fatal("pending checkout price was mutable")
	}
	retry, err := Prepare(ctx, db, 11, 1, "changed@example.test", "https://changed.test")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != retry.ID || first.Email != retry.Email || first.Domain != retry.Domain || first.Description != retry.Description {
		t.Fatal("retry changed provider inputs")
	}
	if _, err = Prepare(ctx, db, 11, 2, "", "domain"); err == nil {
		t.Fatal("other user could prepare checkout")
	}
	if _, err = db.Exec("UPDATE at_booking_people SET person_id=2 WHERE booking_id=11"); err == nil {
		t.Fatal("pending attempt did not freeze participants")
	}
	if err = Attach(ctx, db, first.ID, "recovered", "", first.Amount+1); err == nil {
		t.Fatal("mismatched amount accepted")
	}
	if err = Attach(ctx, db, first.ID, "recovered", "", first.Amount); err != nil {
		t.Fatal(err)
	}
	if err = Attach(ctx, db, first.ID, "recovered", "https://checkout.example.test", first.Amount); err != nil {
		t.Fatal(err)
	}
	if err = Attach(ctx, db, first.ID, "different", "", first.Amount); err == nil {
		t.Fatal("second session accepted")
	}
	if _, err = Finalize(ctx, db, "recovered-event", "recovered", first.Amount, "nzd"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentConflictingAttachmentsHaveOneWinner(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/0006_checkout_attempts.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t, ledgerFixture+`ALTER TABLE at_trips ADD COLUMN description text;UPDATE at_bookings SET stripe_session_id=NULL, from_date='2026-01-02', to_date='2026-01-04' WHERE id=11;DELETE FROM at_checkout_quotes;`+string(migration))
	lifecycle, err := os.ReadFile("../../migrations/0007_checkout_lifecycle.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(lifecycle)); err != nil {
		t.Fatal(err)
	}
	priceGuard, err := os.ReadFile("../../migrations/0009_checkout_price_guard.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(priceGuard)); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	a, err := Prepare(ctx, db, 11, 1, "one@example.test", "https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var successes atomic.Int32
	for _, id := range []string{"first", "second"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if Attach(ctx, db, a.ID, id, "url", a.Amount) == nil {
				successes.Add(1)
			}
		}(id)
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatal("conflicting responses both attached")
	}
	var attemptSession, bookingSession string
	if err = db.Get(&attemptSession, "SELECT session_id FROM at_checkout_attempts WHERE id=$1", a.ID); err != nil {
		t.Fatal(err)
	}
	if err = db.Get(&bookingSession, "SELECT stripe_session_id FROM at_bookings WHERE id=11"); err != nil {
		t.Fatal(err)
	}
	if attemptSession != bookingSession {
		t.Fatal("attempt and booking disagree")
	}
	var quotes int
	if err = db.Get(&quotes, "SELECT count(*) FROM at_checkout_quotes"); err != nil || quotes != 1 {
		t.Fatal(quotes, err)
	}
	if err = Attach(ctx, db, a.ID, attemptSession, "url", a.Amount); err != nil {
		t.Fatal("same response was not retry-safe", err)
	}
}
