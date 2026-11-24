package paymentLedger

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var ErrVoucherUnavailable = errors.New("voucher is invalid or expired")
var ErrQuoteChanged = errors.New("booking total changed")

const activeVoucher = `v.is_active IS TRUE
	AND v.expiry_date IS NOT NULL
	AND ((v.discount_percent IS NULL) <> (v.fixed_cost IS NULL))`

func ValidVoucher(ctx context.Context, tx *sqlx.Tx, code string) (bool, error) {
	var valid bool
	if len(code) > 64 {
		return false, nil
	}
	err := tx.GetContext(ctx, &valid, `SELECT EXISTS(SELECT 1 FROM at_booking_vouchers v WHERE v.code=$1 AND `+activeVoucher+` AND v.expiry_date > (CURRENT_TIMESTAMP AT TIME ZONE 'Pacific/Auckland')::date)`, code)
	return valid, err
}

type BookingReview struct {
	AmountMinor  int64     `json:"amount_minor"`
	VoucherCode  string    `json:"voucher_code"`
	TripName     string    `json:"trip_name"`
	Notes        string    `json:"notes"`
	FromDate     time.Time `json:"from_date"`
	ToDate       time.Time `json:"to_date"`
	Participants []string  `json:"participants"`
}

func bookingTotal(ctx context.Context, tx *sqlx.Tx, bookingID int) (int64, error) {
	return bookingTotalAt(ctx, tx, bookingID, time.Now())
}

func bookingTotalAt(ctx context.Context, tx *sqlx.Tx, bookingID int, effectiveAt time.Time) (int64, error) {
	var row struct {
		Code  string              `db:"voucher_code"`
		Base  decimal.NullDecimal `db:"base"`
		Total decimal.NullDecimal `db:"total"`
	}
	err := tx.GetContext(ctx, &row, `WITH price AS (
		SELECT b.voucher_code, COALESCE(b.booking_price,SUM(c.amount)*(EXTRACT(EPOCH FROM (b.to_date-b.from_date))/86400)) AS base
		FROM at_bookings b JOIN at_trips t ON t.id=b.trip_id JOIN at_booking_people p ON p.booking_id=b.id
		JOIN st_users u ON u.id=p.person_id
		LEFT JOIN at_trip_costs c ON c.trip_cost_group_id=t.trip_cost_group_id AND c.member_status_id=u.member_status_id AND c.user_age_group_id=u.user_age_group_id
		WHERE b.id=$1 GROUP BY b.id HAVING COUNT(p.id)=COUNT(c.id)
	)
	SELECT price.voucher_code, price.base,
		CASE WHEN price.voucher_code='' THEN price.base
		     ELSE COALESCE(v.fixed_cost,price.base*(100-v.discount_percent)/100) END AS total
	FROM price LEFT JOIN at_booking_vouchers v ON v.code=price.voucher_code AND `+activeVoucher+` AND v.expiry_date > ($2::timestamptz AT TIME ZONE 'Pacific/Auckland')::date`, bookingID, effectiveAt)
	if err != nil {
		return 0, err
	}
	if row.Code != "" && !row.Total.Valid {
		return 0, ErrVoucherUnavailable
	}
	if !row.Base.Valid || !row.Total.Valid || row.Total.Decimal.IsNegative() {
		return 0, ErrInvalidPayment
	}
	cents := row.Total.Decimal.Mul(decimal.NewFromInt(100)).Round(0)
	if (cents.IsZero() && row.Code == "") || cents.GreaterThan(decimal.NewFromInt(99999999)) {
		return 0, ErrInvalidPayment
	}
	return cents.IntPart(), nil
}

func ReviewBooking(ctx context.Context, db *sqlx.DB, bookingID, owner int) (BookingReview, error) {
	var review BookingReview
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return review, err
	}
	defer tx.Rollback()
	if err := LockTrip(ctx, tx, bookingID); err != nil {
		return review, err
	}
	err = tx.QueryRowContext(ctx, `SELECT b.voucher_code, t.trip_name, COALESCE(b.notes,''), b.from_date, b.to_date
		FROM at_bookings b JOIN at_trips t ON t.id=b.trip_id
		WHERE b.id=$1 AND b.owner_id=$2 AND b.booking_status_id=1
		AND NOT EXISTS(SELECT 1 FROM at_payments WHERE booking_id=b.id) FOR UPDATE OF b`, bookingID, owner).
		Scan(&review.VoucherCode, &review.TripName, &review.Notes, &review.FromDate, &review.ToDate)
	if err != nil {
		return review, err
	}
	err = tx.SelectContext(ctx, &review.Participants, `SELECT u.name FROM at_booking_people p
		JOIN st_users u ON u.id=p.person_id WHERE p.booking_id=$1 ORDER BY p.id`, bookingID)
	if err != nil {
		return review, err
	}
	err = tx.GetContext(ctx, &review.AmountMinor, `SELECT amount_minor FROM at_checkout_attempts
		WHERE booking_id=$1 AND state IN ('pending','attached') LIMIT 1`, bookingID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := requireParticipantDetails(ctx, tx, bookingID); err != nil {
			return review, err
		}
		review.AmountMinor, err = bookingTotal(ctx, tx, bookingID)
	}
	if err != nil {
		return review, err
	}
	return review, tx.Commit()
}

func ConfirmFreeBooking(ctx context.Context, db *sqlx.DB, bookingID, owner int) error {
	tx, err := Begin(ctx, db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := LockTrip(ctx, tx, bookingID); err != nil {
		return err
	}
	var confirmed bool
	var voucher string
	err = tx.QueryRowContext(ctx, `SELECT free_confirmed_at IS NOT NULL, voucher_code FROM at_bookings b
		WHERE id=$1 AND owner_id=$2 AND (free_confirmed_at IS NOT NULL OR
		(booking_status_id=1 AND stripe_session_id IS NULL
		AND NOT EXISTS(SELECT 1 FROM at_payments WHERE booking_id=b.id)
		AND NOT EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=b.id AND state<>'expired')))
		FOR UPDATE`, bookingID, owner).Scan(&confirmed, &voucher)
	if err != nil {
		return err
	}
	if confirmed {
		return tx.Commit()
	}
	if err := requireParticipantDetails(ctx, tx, bookingID); err != nil {
		return err
	}
	if voucher == "" {
		return ErrQuoteChanged
	}
	voucherID, _, err := lockVoucher(ctx, tx, voucher)
	if err != nil {
		return err
	}
	usedAt := time.Now()
	amount, err := bookingTotalAt(ctx, tx, bookingID, usedAt)
	if err != nil {
		return err
	}
	if amount != 0 {
		return ErrQuoteChanged
	}
	if err := Reserve(ctx, tx, bookingID, time.Now().Add(45*time.Minute)); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE at_bookings SET free_confirmed_at=CURRENT_TIMESTAMP, booking_price=0,
		booking_status_id=3, amount_paid=100, currency='nzd' WHERE id=$1`, bookingID)
	if err != nil {
		return err
	}
	if err := redeemVoucher(ctx, tx, bookingID, voucherID, voucher, "free", "booking-confirmation", usedAt, 0); err != nil {
		return err
	}
	return tx.Commit()
}
