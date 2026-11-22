package paymentLedger

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var ErrInvalidPayment = errors.New("payment does not match the stored checkout quote")

// Finalize commits the event, ledger entry and derived booking state together.
// Row locking serializes duplicate events, including across API replicas.
func Finalize(ctx context.Context, db *sqlx.DB, eventID, sessionID string, amountMinor int64, currency string, paymentTimes ...time.Time) (bool, error) {
	var paidAt time.Time
	if len(paymentTimes) > 0 {
		paidAt = paymentTimes[0]
	}
	replay, err := finalize(ctx, db, eventID, sessionID, amountMinor, currency, paidAt)
	if err != nil && !errors.Is(err, ErrInvalidPayment) {
		// The financial transaction has rolled back. Record a separate durable
		// exception only for a matching stored quote, never untrusted payloads.
		reviewCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if recordErr := RecordReview(reviewCtx, db, sessionID, amountMinor, currency); recordErr != nil {
			return replay, errors.Join(err, recordErr)
		}
	}
	return replay, err
}

func finalize(ctx context.Context, db *sqlx.DB, eventID, sessionID string, amountMinor int64, currency string, paidAt time.Time) (bool, error) {
	if eventID == "" || sessionID == "" || amountMinor <= 0 {
		return false, ErrInvalidPayment
	}
	tx, err := Begin(ctx, db)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var capacityBookingID int
	if err := tx.GetContext(ctx, &capacityBookingID, `SELECT booking_id FROM at_checkout_quotes WHERE session_id=$1`, sessionID); err != nil {
		return false, err
	}
	if err := LockTrip(ctx, tx, capacityBookingID); err != nil {
		return false, err
	}
	var bookingID int
	var expected int64
	var expectedCurrency string
	err = tx.QueryRowContext(ctx, `SELECT b.id,q.amount_minor,q.currency FROM at_checkout_quotes q JOIN at_bookings b ON b.id=q.booking_id WHERE q.session_id=$1 AND b.stripe_session_id=q.session_id FOR UPDATE OF b`, sessionID).Scan(&bookingID, &expected, &expectedCurrency)
	if err != nil {
		return false, err
	}
	if expected != amountMinor || strings.ToLower(currency) != expectedCurrency {
		return false, ErrInvalidPayment
	}
	var refunded bool
	if err := tx.GetContext(ctx, &refunded, `SELECT EXISTS(SELECT 1 FROM at_payment_reviews WHERE session_id=$1 AND resolution='external-refund-confirmed')`, sessionID); err != nil {
		return false, err
	}
	var inserted string
	err = tx.QueryRowContext(ctx, `INSERT INTO at_stripe_events(event_id,session_id) VALUES ($1,$2) ON CONFLICT(event_id) DO NOTHING RETURNING event_id`, eventID, sessionID).Scan(&inserted)
	if err == sql.ErrNoRows {
		var storedSession string
		if err := tx.GetContext(ctx, &storedSession, `SELECT session_id FROM at_stripe_events WHERE event_id=$1`, eventID); err != nil {
			return false, err
		}
		if storedSession != sessionID {
			return false, ErrInvalidPayment
		}
		return true, tx.Commit()
	}
	if err != nil {
		return false, err
	}
	amount := decimal.NewFromInt(amountMinor).Shift(-2)
	if refunded {
		// The operator confirmed a full external refund. Acknowledge delivery
		// durably without admitting the booking or recreating refunded money.
		return true, tx.Commit()
	}
	if err := redeemStripePayment(ctx, tx, bookingID, sessionID, paidAt, amountMinor); err != nil {
		return false, err
	}
	if paidAt.IsZero() {
		paidAt = time.Now()
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO at_payments(booking_id,payment_date,amount,payment_method,stripe_session_id) VALUES($1,$2,$3,'stripe_checkout',$4) ON CONFLICT(stripe_session_id) WHERE stripe_session_id IS NOT NULL DO NOTHING`, bookingID, paidAt, amount, sessionID)
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE at_checkout_attempts SET state='paid' WHERE session_id=$1 AND state<>'expired'`, sessionID); err != nil {
		return false, err
	}
	if err := Sync(ctx, tx, bookingID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE at_payment_reviews SET resolved_at=CURRENT_TIMESTAMP,resolution='admitted',actor='finalization' WHERE session_id=$1 AND resolved_at IS NULL`, sessionID); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

// Quote computes cents from numeric SQL values, never binary floating point.
func Quote(ctx context.Context, tx *sqlx.Tx, bookingID int) (int64, error) {
	amount, err := bookingTotal(ctx, tx, bookingID)
	if err == nil && amount == 0 {
		return 0, ErrQuoteChanged
	}
	return amount, err
}
