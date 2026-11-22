package paymentLedger

import (
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"github.com/guregu/null/v5/zero"
	"github.com/jmoiron/sqlx"
	"time"
)

var ErrValidation = errors.New("invalid financial input")
var ErrLedgerConflict = errors.New("financial transition conflicts with ledger balance")

type PaymentWrite struct {
	BookingID     int          `json:"booking_id"`
	PaymentDate   zero.Time    `json:"payment_date"`
	Amount        models.Money `json:"amount"`
	PaymentMethod zero.String  `json:"payment_method"`
}
type RefundWrite struct {
	PaymentID   int          `json:"payment_id"`
	RefundDate  zero.Time    `json:"refund_date"`
	Amount      models.Money `json:"amount"`
	Reason      zero.String  `json:"reason"`
	ExternalRef zero.String  `json:"external_ref"`
}

func changed(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// WritePayment owns validation, locks, mutation and derived state atomically.
// HTTP adapters supply only mutable input fields and translate typed errors.
func WritePayment(ctx context.Context, db *sqlx.DB, operation string, id int, input PaymentWrite) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if operation != "delete" && (input.BookingID <= 0 || input.Amount.IsNegative()) {
		return 0, ErrValidation
	}
	tx, err := Begin(ctx, db)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var old int
	if operation == "create" {
		err = LockBookings(ctx, tx, input.BookingID)
	} else {
		extra := []int{}
		if operation == "update" {
			extra = append(extra, input.BookingID)
		}
		old, err = LockPayment(ctx, tx, id, extra...)
	}
	if err != nil {
		return 0, err
	}
	if operation != "delete" {
		var confirmedFree bool
		if err = tx.GetContext(ctx, &confirmedFree, `SELECT free_confirmed_at IS NOT NULL OR EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=$1 AND state IN ('pending','attached')) FROM at_bookings WHERE id=$1`, input.BookingID); err != nil {
			return 0, err
		}
		if confirmedFree {
			return 0, ErrLedgerConflict
		}
	}
	var wasPositive bool
	if operation == "update" || operation == "delete" {
		var providerBacked bool
		if err = tx.QueryRowContext(ctx, `SELECT stripe_session_id IS NOT NULL OR EXISTS(SELECT 1 FROM at_voucher_redemptions WHERE source='manual' AND reference=p.id::text AND booking_id=p.booking_id), COALESCE(amount>0,false) FROM at_payments p WHERE id=$1`, id).Scan(&providerBacked, &wasPositive); err != nil {
			return 0, err
		}
		if providerBacked {
			return 0, ErrLedgerConflict
		}
	}
	switch operation {
	case "create":
		err = tx.QueryRowContext(ctx, `INSERT INTO at_payments(booking_id,payment_date,amount,payment_method) VALUES($1,$2,$3,$4) RETURNING id`, input.BookingID, input.PaymentDate, input.Amount, input.PaymentMethod).Scan(&id)
	case "update":
		err = changed(tx.ExecContext(ctx, `UPDATE at_payments SET booking_id=$2,payment_date=$3,amount=$4,payment_method=$5 WHERE id=$1`, id, input.BookingID, input.PaymentDate, input.Amount, input.PaymentMethod))
	case "delete":
		err = changed(tx.ExecContext(ctx, `DELETE FROM at_payments WHERE id=$1`, id))
	default:
		return 0, ErrValidation
	}
	if err != nil {
		return 0, err
	}
	if operation != "delete" {
		if input.Amount.IsPositive() && (operation == "create" || old != input.BookingID || !wasPositive) {
			if err := redeemManualPayment(ctx, tx, input.BookingID, id); err != nil {
				return 0, err
			}
		}
		if err = Sync(ctx, tx, input.BookingID); err != nil {
			return 0, err
		}
	}
	if old > 0 && (operation == "delete" || old != input.BookingID) {
		if err = Sync(ctx, tx, old); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
func WriteRefund(ctx context.Context, db *sqlx.DB, operation string, id int, input RefundWrite) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if operation != "delete" && (input.PaymentID <= 0 || input.Amount.IsNegative()) {
		return 0, ErrValidation
	}
	tx, err := Begin(ctx, db)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var old, newBooking int
	if operation == "create" {
		newBooking, err = LockPayment(ctx, tx, input.PaymentID)
	} else {
		target := input.PaymentID
		if operation == "delete" {
			target = 0
		}
		old, err = LockRefund(ctx, tx, id, target)
	}
	if err != nil {
		return 0, err
	}
	if operation != "delete" {
		newBooking, err = GetBookingIDByPaymentIDTx(ctx, tx, input.PaymentID)
		if err != nil {
			return 0, err
		}
	}
	var providerBacked bool
	if operation == "create" {
		err = tx.GetContext(ctx, &providerBacked, `SELECT stripe_session_id IS NOT NULL OR EXISTS(SELECT 1 FROM at_voucher_redemptions WHERE source='manual' AND reference=p.id::text AND booking_id=p.booking_id) FROM at_payments p WHERE id=$1`, input.PaymentID)
	} else {
		err = tx.GetContext(ctx, &providerBacked, `SELECT provider_refund_id IS NOT NULL FROM at_refunds WHERE id=$1`, id)
	}
	if err != nil {
		return 0, err
	}
	if providerBacked {
		return 0, ErrLedgerConflict
	}
	if operation == "update" {
		if err = tx.GetContext(ctx, &providerBacked, `SELECT stripe_session_id IS NOT NULL OR EXISTS(SELECT 1 FROM at_voucher_redemptions WHERE source='manual' AND reference=p.id::text AND booking_id=p.booking_id) FROM at_payments p WHERE id=$1`, input.PaymentID); err != nil {
			return 0, err
		}
		if providerBacked {
			return 0, ErrLedgerConflict
		}
	}
	switch operation {
	case "create":
		err = tx.QueryRowContext(ctx, `INSERT INTO at_refunds(payment_id,refund_date,amount,reason,external_ref) VALUES($1,$2,$3,$4,$5) RETURNING id`, input.PaymentID, input.RefundDate, input.Amount, input.Reason, input.ExternalRef).Scan(&id)
	case "update":
		err = changed(tx.ExecContext(ctx, `UPDATE at_refunds SET payment_id=$2,refund_date=$3,amount=$4,reason=$5,external_ref=$6 WHERE id=$1`, id, input.PaymentID, input.RefundDate, input.Amount, input.Reason, input.ExternalRef))
	case "delete":
		err = changed(tx.ExecContext(ctx, `DELETE FROM at_refunds WHERE id=$1`, id))
	default:
		return 0, ErrValidation
	}
	if err != nil {
		return 0, err
	}
	if operation != "delete" {
		if err = Sync(ctx, tx, newBooking); err != nil {
			return 0, err
		}
	}
	if old > 0 && (operation == "delete" || old != newBooking) {
		if err = Sync(ctx, tx, old); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
