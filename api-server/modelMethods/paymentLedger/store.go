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
func Finalize(ctx context.Context, db *sqlx.DB, eventID, sessionID string, amountMinor int64, currency string) (bool, error) {
	if eventID == "" || sessionID == "" || amountMinor <= 0 {
		return false, ErrInvalidPayment
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
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
	_, err = tx.ExecContext(ctx, `INSERT INTO at_payments(booking_id,payment_date,amount,payment_method,stripe_session_id) VALUES($1,$2,$3,'stripe_checkout',$4) ON CONFLICT(stripe_session_id) WHERE stripe_session_id IS NOT NULL DO NOTHING`, bookingID, time.Now(), amount, sessionID)
	if err != nil {
		return false, err
	}
	var net decimal.Decimal
	err = tx.GetContext(ctx, &net, `SELECT COALESCE((SELECT SUM(amount) FROM at_payments WHERE booking_id=$1),0)-COALESCE((SELECT SUM(r.amount) FROM at_refunds r JOIN at_payments p ON p.id=r.payment_id WHERE p.booking_id=$1),0)`, bookingID)
	if err != nil {
		return false, err
	}
	status := 1
	pct := int64(0)
	if net.IsPositive() {
		status = 4
		pct = net.Div(amount).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
		if net.GreaterThanOrEqual(amount) {
			status = 3
			pct = 100
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE at_bookings SET booking_status_id=CASE WHEN booking_status_id=2 THEN 2 ELSE $2 END,amount_paid=$3,payment_date=(SELECT MAX(payment_date) FROM at_payments WHERE booking_id=$1) WHERE id=$1`, bookingID, status, pct)
	if err != nil {
		return false, err
	}
	return false, tx.Commit()
}

// Quote computes cents from numeric SQL values, never binary floating point.
func Quote(ctx context.Context, tx *sqlx.Tx, bookingID int) (int64, error) {
	var amount decimal.NullDecimal
	err := tx.GetContext(ctx, &amount, `SELECT COALESCE(b.booking_price,SUM(c.amount)*(EXTRACT(EPOCH FROM (b.to_date-b.from_date))/86400)) FROM at_bookings b JOIN at_trips t ON t.id=b.trip_id JOIN at_booking_people p ON p.booking_id=b.id JOIN st_users u ON u.id=p.person_id LEFT JOIN at_trip_costs c ON c.trip_cost_group_id=t.trip_cost_group_id AND c.member_status_id=u.member_status_id AND c.user_age_group_id=u.user_age_group_id WHERE b.id=$1 GROUP BY b.id HAVING COUNT(p.id)=COUNT(c.id)`, bookingID)
	if err != nil {
		return 0, err
	}
	if !amount.Valid || !amount.Decimal.IsPositive() {
		return 0, ErrInvalidPayment
	}
	cents := amount.Decimal.Mul(decimal.NewFromInt(100)).Round(0)
	if !cents.IsPositive() || cents.GreaterThan(decimal.NewFromInt(99999999)) {
		return 0, ErrInvalidPayment
	}
	return cents.IntPart(), nil
}
