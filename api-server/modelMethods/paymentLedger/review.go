package paymentLedger

import (
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"strings"
)

// RecordReview stores no provider secrets or internal SQL error messages.
func RecordReview(ctx context.Context, db *sqlx.DB, session string, amount int64, currency string) error {
	_, err := db.ExecContext(ctx, `INSERT INTO at_payment_reviews(session_id,booking_id,amount_minor,currency,reason)
 SELECT session_id,booking_id,amount_minor,currency,'finalization_failed' FROM at_checkout_quotes
 WHERE session_id=$1 AND amount_minor=$2 AND currency=$3
 ON CONFLICT(session_id) DO UPDATE SET last_seen=CURRENT_TIMESTAMP,failures=at_payment_reviews.failures+1
 WHERE at_payment_reviews.resolved_at IS NULL`, session, amount, strings.ToLower(currency))
	return err
}

// ResolveReview records an operator-confirmed external refund. It never claims
// to send money. Admission is resolved only by successful ledger finalization.
func ResolveReview(ctx context.Context, db *sqlx.DB, session, actor, evidence string) error {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(evidence) == "" {
		return errors.New("actor and provider refund evidence are required")
	}
	tx, err := Begin(ctx, db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var booking int
	if err = tx.GetContext(ctx, &booking, `SELECT booking_id FROM at_payment_reviews WHERE session_id=$1`, session); err != nil {
		return err
	}
	if err = LockBookings(ctx, tx, booking); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE at_payment_reviews SET resolved_at=CURRENT_TIMESTAMP,resolution='external-refund-confirmed',actor=$2,evidence=$3 WHERE session_id=$1 AND resolved_at IS NULL AND NOT EXISTS(SELECT 1 FROM at_payments WHERE stripe_session_id=$1)`, session, actor, evidence)
	if err = changed(result, err); err != nil {
		return err
	}
	return tx.Commit()
}
