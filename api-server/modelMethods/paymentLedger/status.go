package paymentLedger

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"time"
)

const qryBookingPaymentSummary = `WITH booking_costs AS (
		SELECT
			atb.id,
			atb.booking_status_id,
			COALESCE(atb.booking_price, 0) AS booking_price,
			COALESCE(
				SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400),
				0
			) AS booking_cost
		FROM at_bookings atb
		JOIN at_trips att ON att.id = atb.trip_id
		LEFT JOIN at_booking_people atbp ON atbp.booking_id = atb.id
		LEFT JOIN st_users stu ON stu.id = atbp.person_id
		LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id = att.trip_cost_group_id
			AND attc.member_status_id = stu.member_status_id
			AND attc.user_age_group_id = stu.user_age_group_id
		WHERE atb.id = $1
		GROUP BY atb.id, atb.booking_status_id, atb.booking_price
	), payment_totals AS (
		SELECT
			booking_id,
			COALESCE(SUM(amount), 0) AS total_paid,
			MAX(payment_date) AS latest_payment_date
		FROM at_payments
		WHERE booking_id = $1
		GROUP BY booking_id
	), refund_totals AS (
		SELECT
			p.booking_id,
			COALESCE(SUM(r.amount), 0) AS total_refunded
		FROM at_refunds r
		JOIN at_payments p ON p.id = r.payment_id
		WHERE p.booking_id = $1
		GROUP BY p.booking_id
	)
	SELECT
		bc.booking_status_id,
		bc.booking_price,
		bc.booking_cost,
		COALESCE(pt.total_paid, 0) AS total_paid,
		COALESCE(rt.total_refunded, 0) AS total_refunded,
		pt.latest_payment_date
	FROM booking_costs bc
	LEFT JOIN payment_totals pt ON pt.booking_id = bc.id
	LEFT JOIN refund_totals rt ON rt.booking_id = bc.id`

const qryUpdateBookingPaymentStatus = `UPDATE at_bookings
	SET booking_status_id = $2,
		amount_paid = $3,
		payment_date = $4
	WHERE id = $1`

const qryGetBookingIDByPaymentID = `SELECT booking_id FROM at_payments WHERE id = $1`
const qryGetBookingIDByRefundID = `SELECT p.booking_id
	FROM at_refunds r
	JOIN at_payments p ON p.id = r.payment_id
	WHERE r.id = $1`

type bookingPaymentSummary struct {
	BookingStatusID   int             `db:"booking_status_id"`
	BookingPrice      decimal.Decimal `db:"booking_price"`
	BookingCost       decimal.Decimal `db:"booking_cost"`
	TotalPaid         decimal.Decimal `db:"total_paid"`
	TotalRefunded     decimal.Decimal `db:"total_refunded"`
	LatestPaymentDate sql.NullTime    `db:"latest_payment_date"`
}

func GetBookingIDByPaymentIDTx(ctx context.Context, tx *sqlx.Tx, paymentID int) (int, error) {
	var bookingID int
	if err := tx.GetContext(ctx, &bookingID, qryGetBookingIDByPaymentID, paymentID); err != nil {
		return 0, err
	}
	return bookingID, nil
}

func GetBookingIDByRefundIDTx(ctx context.Context, tx *sqlx.Tx, refundID int) (int, error) {
	var bookingID int
	if err := tx.GetContext(ctx, &bookingID, qryGetBookingIDByRefundID, refundID); err != nil {
		return 0, err
	}
	return bookingID, nil
}

func Sync(ctx context.Context, tx *sqlx.Tx, bookingID int) error {
	if err := LockTrip(ctx, tx, bookingID); err != nil {
		return err
	}
	var locked int
	if err := tx.GetContext(ctx, &locked, `SELECT id FROM at_bookings WHERE id=$1 FOR UPDATE`, bookingID); err != nil {
		return err
	}
	var invalid bool
	if err := tx.GetContext(ctx, &invalid, `SELECT EXISTS(SELECT 1 FROM at_payments p WHERE p.booking_id=$1 AND (p.amount<0 OR p.amount<COALESCE((SELECT SUM(r.amount) FROM at_refunds r WHERE r.payment_id=p.id),0))) OR EXISTS(SELECT 1 FROM at_refunds r JOIN at_payments p ON p.id=r.payment_id WHERE p.booking_id=$1 AND r.amount<0)`, bookingID); err != nil {
		return err
	}
	if invalid {
		return errors.New("refund exceeds payment or ledger amount is negative")
	}
	var summary bookingPaymentSummary
	if err := tx.GetContext(ctx, &summary, qryBookingPaymentSummary, bookingID); err != nil {
		return err
	}
	net := summary.TotalPaid.Sub(summary.TotalRefunded)
	target := summary.BookingPrice
	if !target.IsPositive() {
		target = summary.BookingCost
	}
	status, pct := 1, int64(0)
	var paidAt any
	if net.IsPositive() {
		status = 4
		if target.IsPositive() {
			pct = net.Div(target).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
			if net.GreaterThanOrEqual(target) {
				status = 3
				pct = 100
			}
		}
		if summary.LatestPaymentDate.Valid {
			paidAt = summary.LatestPaymentDate.Time
		}
	}
	if summary.BookingStatusID == 2 {
		status = 2
	}
	if status == 3 || status == 4 {
		if err := Reserve(ctx, tx, bookingID, time.Now().Add(45*time.Minute)); err != nil {
			return err
		}
	}
	if status == 2 || status == 1 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM at_checkout_reservations r WHERE r.booking_id=$1 AND ($2=2 OR NOT EXISTS(SELECT 1 FROM at_checkout_attempts a WHERE a.booking_id=$1 AND a.state IN ('pending','attached')))`, bookingID, status); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, qryUpdateBookingPaymentStatus, bookingID, status, pct, paidAt)
	return err
}

// Begin leaves lock scope to the affected-trip/booking locking helpers. Provider
// operations do not hold these transactions; unrelated trips remain independent.
func Begin(ctx context.Context, db *sqlx.DB) (*sqlx.Tx, error) { return db.BeginTxx(ctx, nil) }
