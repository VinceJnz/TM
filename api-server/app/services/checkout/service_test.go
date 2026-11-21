package checkout

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"os"
	"testing"
	"time"
)

type fakeProvider struct {
	sessions  []Session
	lookups   int
	createErr bool
	db        *sqlx.DB
}

func (f *fakeProvider) Create(ctx context.Context, a paymentLedger.Attempt) (Session, error) {
	// Provider requests must not run with a booking lock held by the service.
	tx, err := f.db.BeginTxx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback()
	var id int
	if err = tx.GetContext(ctx, &id, "SELECT id FROM at_bookings WHERE id=$1 FOR UPDATE NOWAIT", a.BookingID); err != nil {
		return Session{}, err
	}
	result := Session{ID: "session-" + a.ID, URL: "https://provider.example.test", AttemptID: a.ID, Amount: a.Amount, Currency: "nzd", Mode: "payment", State: "open"}
	f.sessions = []Session{result}
	if f.createErr {
		return Session{}, errors.New("synthetic lost response")
	}
	return result, nil
}
func (f *fakeProvider) Lookup(ctx context.Context, a paymentLedger.Attempt) ([]Session, error) {
	f.lookups++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return f.sessions, nil
}
func recoveryDB(t *testing.T) *sqlx.DB {
	t.Helper()
	schema := testdb.Ledger + `ALTER TABLE at_trips ADD COLUMN description text;UPDATE at_bookings SET stripe_session_id=NULL WHERE id=11;DELETE FROM at_checkout_quotes;`
	for _, name := range []string{"0006_checkout_attempts.sql", "0007_checkout_lifecycle.sql", "0009_checkout_price_guard.sql"} {
		data, err := os.ReadFile("../../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		schema += string(data)
	}
	return testdb.Open(t, schema)
}
func TestExpiredLostResponseReleasesInputsAndAllowsReplacement(t *testing.T) {
	db := recoveryDB(t)
	provider := &fakeProvider{db: db, createErr: true}
	service := New(db, provider)
	ctx := context.Background()
	if _, err := service.Start(ctx, 11, 1, "member@example.test", "https://app.test"); err == nil {
		t.Fatal("expected lost response")
	}
	old := provider.sessions[0].AttemptID
	if _, err := service.Reconcile(ctx, 11, 2); err == nil || provider.lookups != 0 {
		t.Fatal("other owner could inspect provider checkout")
	}
	provider.sessions[0].State = "expired"
	if result, err := service.Reconcile(ctx, 11, 1); err != nil || result != "expired" {
		t.Fatal(result, err)
	}
	var state string
	if err := db.Get(&state, "SELECT state FROM at_checkout_attempts WHERE id=$1", old); err != nil || state != "expired" {
		t.Fatal(state, err)
	}
	if _, err := db.Exec("UPDATE at_booking_people SET notes='editable' WHERE booking_id=11"); err != nil {
		t.Fatal("expired history still froze inputs", err)
	}
	provider.createErr = false
	result, err := service.Start(ctx, 11, 1, "member@example.test", "https://app.test")
	if err != nil || result.AttemptID == old {
		t.Fatal("replacement failed", result, err)
	}
	var count int
	if err = db.Get(&count, "SELECT count(*) FROM at_checkout_attempts"); err != nil || count != 2 {
		t.Fatal("history was not retained", err)
	}
}
func TestPaidRecoveryAndUncertainAbsence(t *testing.T) {
	db := recoveryDB(t)
	provider := &fakeProvider{db: db, createErr: true}
	service := New(db, provider)
	ctx := context.Background()
	service.Start(ctx, 11, 1, "member@example.test", "https://app.test")
	provider.sessions[0].State = "complete"
	provider.sessions[0].Paid = true
	if result, err := service.Reconcile(ctx, 11, 1); err != nil || result != "paid" {
		t.Fatal(result, err)
	}
	var status int
	if err := db.Get(&status, "SELECT booking_status_id FROM at_bookings WHERE id=11"); err != nil || status != 3 {
		t.Fatal(status, err)
	}
	db = recoveryDB(t)
	provider = &fakeProvider{db: db, createErr: true}
	service = New(db, provider)
	service.Start(ctx, 11, 1, "member@example.test", "https://app.test")
	provider.sessions = nil
	if _, err := service.Reconcile(ctx, 11, 1); err == nil {
		t.Fatal("absence before expiry released inputs")
	}
	service.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if outcome, err := service.Reconcile(ctx, 11, 1); err != nil || outcome != "expired" {
		t.Fatal(outcome, err)
	}
}

func TestProviderSuccessThenPersistenceFailureCanRecover(t *testing.T) {
	db := recoveryDB(t)
	provider := &fakeProvider{db: db}
	service := New(db, provider)
	ctx := context.Background()
	if _, err := db.Exec(`CREATE FUNCTION fail_quote() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic persistence failure';END $$;CREATE TRIGGER fail_quote BEFORE INSERT ON at_checkout_quotes FOR EACH ROW EXECUTE FUNCTION fail_quote()`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(ctx, 11, 1, "member@example.test", "https://app.test"); err == nil {
		t.Fatal("persistence failure was swallowed")
	}
	if _, err := db.Exec("DROP TRIGGER fail_quote ON at_checkout_quotes"); err != nil {
		t.Fatal(err)
	}
	provider.sessions[0].State = "complete"
	provider.sessions[0].Paid = true
	if outcome, err := service.Reconcile(ctx, 11, 1); err != nil || outcome != "paid" {
		t.Fatal(outcome, err)
	}
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM at_payments"); err != nil || count != 1 {
		t.Fatal(count, err)
	}
}
