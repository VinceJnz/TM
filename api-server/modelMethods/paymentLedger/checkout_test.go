package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"os"
	"testing"
)

func TestDurableAttemptRetriesAndWebhookRecovery(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/0006_checkout_attempts.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t, ledgerFixture+`ALTER TABLE at_trips ADD COLUMN description text;UPDATE at_bookings SET stripe_session_id=NULL WHERE id=11;DELETE FROM at_checkout_quotes;`+string(migration))
	ctx := context.Background()
	first, err := Prepare(ctx, db, 11, 1, "one@example.test", "https://example.test/api/v1")
	if err != nil {
		t.Fatal(err)
	}
	retry, err := Prepare(ctx, db, 11, 1, "changed@example.test", "https://changed.test")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != retry.ID || first.Email != retry.Email || first.Domain != retry.Domain {
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
