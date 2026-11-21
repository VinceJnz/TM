package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"testing"
)

func TestFailedPaidFinalizationCreatesDurableReviewAndRetryResolves(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`UPDATE at_trips SET max_participants=0;`)
	ctx := context.Background()
	for _, event := range []string{"failed-a", "failed-b"} {
		if _, err := Finalize(ctx, db, event, "checkout", 1000, "nzd"); err == nil {
			t.Fatal("expected capacity failure")
		}
	}
	var failures int
	if err := db.Get(&failures, `SELECT failures FROM at_payment_reviews WHERE session_id='checkout' AND resolved_at IS NULL`); err != nil || failures != 2 {
		t.Fatal(failures, err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM at_payments`)
	if count != 0 {
		t.Fatal("failed finalization recorded money")
	}
	if _, err := db.Exec(`UPDATE at_trips SET max_participants=10`); err != nil {
		t.Fatal(err)
	}
	if _, err := Finalize(ctx, db, "retry", "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
	db.Get(&count, `SELECT count(*) FROM at_payment_reviews WHERE resolved_at IS NULL`)
	if count != 0 {
		t.Fatal("retry left review open")
	}
}

func TestReviewRejectsUnmatchedQuotesAndRequiresRefundEvidence(t *testing.T) {
	db := testdb.Open(t, ledgerFixture)
	ctx := context.Background()
	if err := RecordReview(ctx, db, "checkout", 999, "nzd"); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM at_payment_reviews`)
	if count != 0 {
		t.Fatal("unmatched quote persisted")
	}
	if err := RecordReview(ctx, db, "checkout", 1000, "nzd"); err != nil {
		t.Fatal(err)
	}
	if err := ResolveReview(ctx, db, "checkout", "operator", ""); err == nil {
		t.Fatal("missing evidence accepted")
	}
	if err := ResolveReview(ctx, db, "checkout", "operator", "re_synthetic: verified full refund"); err != nil {
		t.Fatal(err)
	}
	if err := ResolveReview(ctx, db, "checkout", "other", "overwrite"); err == nil {
		t.Fatal("resolution was overwritten")
	}
	if _, err := Finalize(ctx, db, "late-event", "checkout", 1000, "nzd"); err == nil {
		t.Fatal("refunded exception admitted a late event")
	}
}
