package checkout

import (
	"context"
	"testing"
)

func TestWorkerRecoversAbandonedPaidSessionAndHonorsLease(t *testing.T) {
	db := recoveryDB(t)
	provider := &fakeProvider{db: db, createErr: true}
	service := New(db, provider)
	ctx := context.Background()
	service.Start(ctx, 11, 1, "member@example.test", "https://app.test")
	provider.sessions[0].State = "complete"
	provider.sessions[0].Paid = true
	if _, err := db.Exec(`UPDATE at_checkout_attempts SET expires_at=CURRENT_TIMESTAMP+interval '30 minutes',reconcile_after=CURRENT_TIMESTAMP+interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	if err := service.RecoverBatch(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if provider.lookups != 0 {
		t.Fatal("claimed a leased attempt")
	}
	if _, err := db.Exec(`UPDATE at_checkout_attempts SET reconcile_after=CURRENT_TIMESTAMP-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	if err := service.RecoverBatch(ctx, 1); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM at_payments`); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if err := service.RecoverBatch(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if provider.lookups != 1 {
		t.Fatal("paid attempt processed twice")
	}
}

func TestWorkerPersistsBackoffAndStopsOnCancellation(t *testing.T) {
	db := recoveryDB(t)
	provider := &fakeProvider{db: db, createErr: true}
	service := New(db, provider)
	ctx := context.Background()
	service.Start(ctx, 11, 1, "member@example.test", "https://app.test")
	provider.sessions[0].Amount = 1
	db.Exec(`UPDATE at_checkout_attempts SET expires_at=CURRENT_TIMESTAMP+interval '30 minutes'`)
	if err := service.RecoverBatch(ctx, 1); err != nil {
		t.Fatal(err)
	}
	var failures int
	if err := db.Get(&failures, `SELECT reconcile_failures FROM at_checkout_attempts`); err != nil || failures != 1 {
		t.Fatal(failures, err)
	}
	if err := service.RecoverBatch(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if provider.lookups != 1 {
		t.Fatal("ignored retry schedule")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := service.RecoverBatch(canceled, 1); err == nil {
		t.Fatal("ignored cancellation")
	}
}
