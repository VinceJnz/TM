package helpers

import (
	"api-server/v2/models"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
)

const qryBookingPaymentSummary = `WITH booking_costs AS (
		SELECT
			atb.id,
			atb.booking_status_id,
			COALESCE(atb.booking_price::double precision, 0) AS booking_price,
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
	BookingStatusID   int          `db:"booking_status_id"`
	BookingPrice      float64      `db:"booking_price"`
	BookingCost       float64      `db:"booking_cost"`
	TotalPaid         float64      `db:"total_paid"`
	TotalRefunded     float64      `db:"total_refunded"`
	LatestPaymentDate sql.NullTime `db:"latest_payment_date"`
}

func GetBookingIDByPaymentIDTx(tx *sqlx.Tx, paymentID int) (int, error) {
	var bookingID int
	if err := tx.Get(&bookingID, qryGetBookingIDByPaymentID, paymentID); err != nil {
		return 0, err
	}
	return bookingID, nil
}

func GetBookingIDByRefundIDTx(tx *sqlx.Tx, refundID int) (int, error) {
	var bookingID int
	if err := tx.Get(&bookingID, qryGetBookingIDByRefundID, refundID); err != nil {
		return 0, err
	}
	return bookingID, nil
}

func SyncBookingPaymentStatusTx(tx *sqlx.Tx, bookingID int) error {
	if bookingID <= 0 {
		return fmt.Errorf("invalid booking id")
	}

	var summary bookingPaymentSummary
	if err := tx.Get(&summary, qryBookingPaymentSummary, bookingID); err != nil {
		return err
	}

	netPaid := summary.TotalPaid - summary.TotalRefunded
	if netPaid < 0 {
		netPaid = 0
	}

	targetAmount := summary.BookingPrice
	if targetAmount <= 0 {
		targetAmount = summary.BookingCost
	}

	statusID := int(models.Not_paid)
	amountPaidPct := int64(0)
	var paymentDate any

	if summary.BookingStatusID == int(models.Cancelled) {
		statusID = summary.BookingStatusID
	} else if targetAmount > 0 {
		paidRatio := netPaid / targetAmount
		if paidRatio < 0 {
			paidRatio = 0
		}
		if paidRatio > 1 {
			paidRatio = 1
		}
		amountPaidPct = int64(math.Round(paidRatio * 100))

		switch {
		case netPaid <= 0.000001:
			statusID = int(models.Not_paid)
		case netPaid >= targetAmount-0.005:
			statusID = int(models.Full_amountPaid)
		default:
			statusID = int(models.Partial_amountPaid)
		}
	} else if netPaid > 0.000001 {
		statusID = int(models.Partial_amountPaid)
	}

	if netPaid > 0.000001 && summary.LatestPaymentDate.Valid {
		paymentDate = summary.LatestPaymentDate.Time
	} else {
		paymentDate = nil
	}

	_, err := tx.Exec(qryUpdateBookingPaymentStatus, bookingID, statusID, amountPaidPct, paymentDate)
	return err
}

func TimeOrNil(validTime sql.NullTime) any {
	if validTime.Valid {
		return validTime.Time
	}
	return nil
}

func MaxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
