package paymentLedger

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
)

func lockVoucher(ctx context.Context, tx *sqlx.Tx, code string) (int, time.Time, error) {
	var id int
	var cutoff time.Time
	err := tx.QueryRowContext(ctx, `SELECT id, expiry_date::timestamp AT TIME ZONE 'Pacific/Auckland' FROM at_booking_vouchers v WHERE code=$1 AND `+activeVoucher+` FOR SHARE`, code).Scan(&id, &cutoff)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrVoucherUnavailable
	}
	return id, cutoff, err
}

func redeemVoucher(ctx context.Context, tx *sqlx.Tx, bookingID, voucherID int, code, source, reference string, usedAt time.Time, amount int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO at_voucher_redemptions(booking_id,voucher_id,voucher_code,source,reference,used_at,amount_minor)
		VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(booking_id) DO NOTHING`, bookingID, voucherID, code, source, reference, usedAt, amount)
	return err
}

func redeemManualPayment(ctx context.Context, tx *sqlx.Tx, bookingID, paymentID int) error {
	var code string
	err := tx.GetContext(ctx, &code, `SELECT voucher_code FROM at_bookings b WHERE id=$1 AND voucher_code<>''
		AND NOT EXISTS(SELECT 1 FROM at_voucher_redemptions WHERE booking_id=b.id)
		AND NOT EXISTS(SELECT 1 FROM at_payments WHERE booking_id=b.id AND id<>$2 AND amount>0)`, bookingID, paymentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var paidAt sql.NullTime
	if err := tx.GetContext(ctx, &paidAt, `SELECT payment_date::timestamp AT TIME ZONE 'Pacific/Auckland' FROM at_payments WHERE id=$1`, paymentID); err != nil {
		return err
	}
	if !paidAt.Valid {
		return ErrValidation
	}
	voucherID, _, err := lockVoucher(ctx, tx, code)
	if err != nil {
		return err
	}
	amount, err := bookingTotalAt(ctx, tx, bookingID, paidAt.Time)
	if err != nil {
		return err
	}
	if amount == 0 {
		return ErrLedgerConflict
	}
	return redeemVoucher(ctx, tx, bookingID, voucherID, code, "manual", strconv.Itoa(paymentID), paidAt.Time, amount)
}

func redeemStripePayment(ctx context.Context, tx *sqlx.Tx, bookingID int, sessionID string, paidAt time.Time, amount int64) error {
	var code string
	var cutoff sql.NullTime
	var voucherID sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT b.voucher_code, a.voucher_id, a.voucher_cutoff FROM at_bookings b
		LEFT JOIN at_checkout_attempts a ON a.booking_id=b.id AND a.session_id=$2 WHERE b.id=$1`, bookingID, sessionID).
		Scan(&code, &voucherID, &cutoff)
	if err != nil || code == "" {
		return err
	}
	var alreadyPaid bool
	if err := tx.GetContext(ctx, &alreadyPaid, `SELECT EXISTS(SELECT 1 FROM at_payments WHERE stripe_session_id=$1)`, sessionID); err != nil {
		return err
	}
	if alreadyPaid {
		return nil
	}
	if paidAt.IsZero() || !voucherID.Valid || !cutoff.Valid || !paidAt.Before(cutoff.Time) {
		return ErrVoucherUnavailable
	}
	return redeemVoucher(ctx, tx, bookingID, int(voucherID.Int64), code, "stripe", sessionID, paidAt, amount)
}
