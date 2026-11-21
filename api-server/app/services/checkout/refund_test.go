package checkout

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"os"
	"testing"
)

type fakeRefund struct{ result Refund }

func (f *fakeRefund) LookupRefund(context.Context, string, string) (Refund, error) {
	return f.result, nil
}

func TestProviderRefundOnlySucceededCountsAndReplayIsImmutable(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/0012_provider_refunds.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t, testdb.Ledger+`ALTER TABLE at_refunds ADD COLUMN refund_date timestamptz,ADD COLUMN reason text,ADD COLUMN external_ref text;`+string(migration))
	ctx := context.Background()
	if _, err = paymentLedger.Finalize(ctx, db, "paid", "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
	var payment int
	db.Get(&payment, `SELECT id FROM at_payments`)
	provider := &fakeRefund{Refund{ID: "re_synthetic", SessionID: "checkout", Currency: "nzd", Status: "pending", Amount: 500}}
	if err = ConfirmRefund(ctx, db, provider, payment, "re_synthetic", "operator"); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM at_refunds`)
	if count != 0 {
		t.Fatal("pending refund reduced balance")
	}
	provider.result.Status = "failed"
	if err = ConfirmRefund(ctx, db, provider, payment, "re_synthetic", "operator"); err != nil {
		t.Fatal(err)
	}
	provider.result.Status = "succeeded"
	for i := 0; i < 2; i++ {
		if err = ConfirmRefund(ctx, db, provider, payment, "re_synthetic", "operator"); err != nil {
			t.Fatal(err)
		}
	}
	db.Get(&count, `SELECT count(*) FROM at_refunds`)
	if count != 1 {
		t.Fatal("duplicate refund")
	}
	var id int
	db.Get(&id, `SELECT id FROM at_refunds`)
	if _, err = paymentLedger.WriteRefund(ctx, db, "delete", id, paymentLedger.RefundWrite{}); err == nil {
		t.Fatal("provider refund deleted")
	}
	if _, err = paymentLedger.WriteRefund(ctx, db, "create", 0, paymentLedger.RefundWrite{PaymentID: payment}); err == nil {
		t.Fatal("manual refund accepted for provider payment")
	}
	provider.result.Status = "pending"
	if err = ConfirmRefund(ctx, db, provider, payment, "re_synthetic", "operator"); err == nil {
		t.Fatal("successful refund regressed")
	}
	provider.result.Status = "succeeded"
	provider.result.SessionID = "another-session"
	if err = ConfirmRefund(ctx, db, provider, payment, "re_synthetic", "operator"); err == nil {
		t.Fatal("wrong payment accepted")
	}
}
