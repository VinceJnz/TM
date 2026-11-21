package paymentLedger

import (
	"context"
	"github.com/jmoiron/sqlx"
)

func ActiveAttempt(ctx context.Context, db *sqlx.DB, bookingID, owner int) (Attempt, error) {
	var a Attempt
	err := db.GetContext(ctx, &a, `SELECT `+qualifiedAttemptColumns+` FROM at_checkout_attempts a JOIN at_bookings b ON b.id=a.booking_id WHERE b.id=$1 AND b.owner_id=$2 AND a.state IN ('pending','attached')`, bookingID, owner)
	return a, err
}

const qualifiedAttemptColumns = `a.id,a.booking_id,a.amount_minor,a.email,a.title,a.description,a.domain,a.expires_at,a.session_id,a.checkout_url,a.state`

// Expire is called only after an authoritative provider lookup. The supplied
// session must still match the locked row, and no money may have been recorded.
func Expire(ctx context.Context, db *sqlx.DB, key string, owner int, confirmedSession string) error {
	tx, err := Begin(ctx, db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var a Attempt
	if err = tx.GetContext(ctx, &a, `SELECT `+attemptColumns+` FROM at_checkout_attempts WHERE id=$1`, key); err != nil {
		return err
	}
	if err = LockTrip(ctx, tx, a.BookingID); err != nil {
		return err
	}
	var id int
	if err = tx.GetContext(ctx, &id, `SELECT id FROM at_bookings WHERE id=$1 AND owner_id=$2 AND booking_status_id=1 AND NOT EXISTS(SELECT 1 FROM at_payments WHERE booking_id=$1) FOR UPDATE`, a.BookingID, owner); err != nil {
		return ErrCheckoutUnavailable
	}
	if err = tx.GetContext(ctx, &a, `SELECT `+attemptColumns+` FROM at_checkout_attempts WHERE id=$1 FOR UPDATE`, key); err != nil {
		return err
	}
	if a.State == "expired" {
		return tx.Commit()
	}
	if a.State == "paid" || a.SessionID.String != confirmedSession {
		return ErrCheckoutUnavailable
	}
	if _, err = tx.ExecContext(ctx, `UPDATE at_checkout_attempts SET state='expired' WHERE id=$1`, key); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE at_bookings SET stripe_session_id=NULL,booking_price=NULL WHERE id=$1 AND (stripe_session_id IS NULL OR stripe_session_id=$2)`, a.BookingID, confirmedSession); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM at_checkout_reservations WHERE booking_id=$1`, a.BookingID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO at_checkout_reconciliations(attempt_id,actor_id,outcome) VALUES($1,$2,'provider-confirmed-expired')`, key, owner); err != nil {
		return err
	}
	return tx.Commit()
}
