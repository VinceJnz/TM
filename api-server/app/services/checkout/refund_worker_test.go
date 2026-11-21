package checkout

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"errors"
	"os"
	"testing"
)

type recoveryRefundProvider struct {
	result Refund
	calls  int
	fail   bool
}

func (f *recoveryRefundProvider) LookupRefund(ctx context.Context, session, id string) (Refund, error) {
	f.calls++
	if err := ctx.Err(); err != nil {
		return Refund{}, err
	}
	if f.fail {
		return Refund{}, errors.New("synthetic provider outage")
	}
	return f.result, nil
}
func TestRefundWorkerRetriesPendingObservationsWithoutDuplicateCredits(t *testing.T) {
	schema := testdb.Ledger + `ALTER TABLE at_refunds ADD COLUMN refund_date timestamptz,ADD COLUMN reason text,ADD COLUMN external_ref text;`
	for _, name := range []string{"0012_provider_refunds.sql", "0014_refund_recovery.sql"} {
		data, err := os.ReadFile("../../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		schema += string(data)
	}
	db := testdb.Open(t, schema)
	ctx := context.Background()
	if _, err := paymentLedger.Finalize(ctx, db, "paid", "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
	var payment int
	db.Get(&payment, `SELECT id FROM at_payments`)
	provider := &recoveryRefundProvider{result: Refund{ID: "re_recovery", SessionID: "checkout", Currency: "nzd", Status: "pending", Amount: 500}}
	if err := ConfirmRefund(ctx, db, provider, payment, "re_recovery", "operator"); err != nil {
		t.Fatal(err)
	}
	provider.fail = true
	if err := RecoverRefundBatch(ctx, db, provider, 1); err != nil {
		t.Fatal(err)
	}
	calls := provider.calls
	if err := RecoverRefundBatch(ctx, db, provider, 1); err != nil || provider.calls != calls {
		t.Fatal("ignored durable backoff", err)
	}
	var failures int
	if err := db.Get(&failures, `SELECT reconcile_failures FROM at_provider_refunds`); err != nil || failures != 1 {
		t.Fatal(failures, err)
	}
	provider.fail = false
	provider.result.Status = "succeeded"
	if _, err := db.Exec(`UPDATE at_provider_refunds SET reconcile_after=CURRENT_TIMESTAMP-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	if err := RecoverRefundBatch(ctx, db, provider, 1); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM at_refunds`)
	if count != 1 {
		t.Fatal("confirmed refund was not posted", count)
	}
	calls = provider.calls
	if err := RecoverRefundBatch(ctx, db, provider, 1); err != nil || provider.calls != calls {
		t.Fatal("terminal refund retried", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := RecoverRefundBatch(canceled, db, provider, 1); err == nil {
		t.Fatal("ignored cancellation")
	}
}
