package finance

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"os"
	"testing"
)

func TestManualChangesRetainActorBeforeAfterAndRollbackAtomically(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/0013_financial_audit.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t, testdb.Ledger+string(migration))
	service := Service{DB: db}
	ctx := context.Background()
	var input paymentLedger.PaymentWrite
	input.BookingID = 11
	if err = input.Amount.UnmarshalJSON([]byte("10")); err != nil {
		t.Fatal(err)
	}
	id, err := service.Payment(ctx, 42, "cash received", "create", 0, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Payment(ctx, 42, "", "update", id, input); err == nil {
		t.Fatal("missing reason accepted")
	}
	if err = input.Amount.UnmarshalJSON([]byte("9")); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Payment(ctx, 42, "correct receipt amount", "update", id, input); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Payment(ctx, 42, "duplicate entry", "delete", id, input); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.Get(&count, `SELECT count(*) FROM at_financial_audit WHERE actor='user:42' AND reason<>''`); err != nil || count != 3 {
		t.Fatal(count, err)
	}
	var before, after string
	if err = db.QueryRow(`SELECT before_data->>'amount',after_data->>'amount' FROM at_financial_audit WHERE operation='UPDATE'`).Scan(&before, &after); err != nil || before != "10" || after != "9" {
		t.Fatal(before, after, err)
	}
	for _, sql := range []string{`DELETE FROM at_financial_audit`, `UPDATE at_financial_audit SET reason='rewrite'`, `TRUNCATE at_financial_audit`} {
		if _, err = db.Exec(sql); err == nil {
			t.Fatal("audit history mutable")
		}
	}
	// Force the derived-state update to fail: neither payment nor audit may survive.
	if _, err = db.Exec(`CREATE FUNCTION reject_sync() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic failure'; END $$;CREATE TRIGGER fail_sync BEFORE UPDATE ON at_bookings FOR EACH ROW EXECUTE FUNCTION reject_sync();`); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Payment(ctx, 42, "cash received", "create", 0, input); err == nil {
		t.Fatal("failed transition succeeded")
	}
	if err = db.Get(&count, `SELECT count(*) FROM at_financial_audit`); err != nil || count != 3 {
		t.Fatal("failed transaction left audit event", count, err)
	}
}
