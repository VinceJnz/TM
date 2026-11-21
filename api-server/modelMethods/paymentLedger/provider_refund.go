package paymentLedger

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type ProviderRefund struct {
	ID, SessionID, Currency, Status string
	Amount                          int64
}

func ApplyProviderRefund(ctx context.Context, db *sqlx.DB, payment int, actor string, refund ProviderRefund) error {
	if payment <= 0 || refund.ID == "" || refund.SessionID == "" || refund.Currency != "nzd" || refund.Amount <= 0 {
		return ErrInvalidPayment
	}
	switch refund.Status {
	case "pending", "requires_action", "succeeded", "failed", "canceled":
	default:
		return ErrValidation
	}
	refundID := refund.ID
	tx, err := Begin(WithAudit(ctx, "operator:"+actor, "provider refund "+refundID), db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	booking, err := LockPayment(ctx, tx, payment)
	if err != nil {
		return err
	}
	var session string
	if err = tx.GetContext(ctx, &session, `SELECT stripe_session_id FROM at_payments WHERE id=$1`, payment); err != nil {
		return err
	}
	if session != refund.SessionID {
		return ErrInvalidPayment
	}
	var oldStatus string
	if err = tx.GetContext(ctx, &oldStatus, `SELECT status FROM at_provider_refunds WHERE id=$1 AND payment_id=$2 FOR UPDATE`, refundID, payment); err != nil {
		return err
	}
	if oldStatus == "succeeded" && refund.Status != "succeeded" {
		return ErrLedgerConflict
	}
	if oldStatus == "succeeded" {
		var oldAmount int64
		if err = tx.GetContext(ctx, &oldAmount, `SELECT amount_minor FROM at_provider_refunds WHERE id=$1`, refundID); err != nil {
			return err
		}
		if oldAmount != refund.Amount {
			return ErrLedgerConflict
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE at_provider_refunds SET status=$2,amount_minor=$3,checked_at=CURRENT_TIMESTAMP WHERE id=$1`, refundID, refund.Status, refund.Amount); err != nil {
		return err
	}
	if refund.Status == "succeeded" {
		// An operator may have recorded this refund before provider tracking
		// existed. Bind a unique, exact historical entry instead of crediting
		// the same refund twice. Ambiguous history requires explicit review.
		var legacy []struct {
			ID        int
			PaymentID int `db:"payment_id"`
			Amount    decimal.Decimal
		}
		if err = tx.SelectContext(ctx, &legacy, `SELECT id,payment_id,amount FROM at_refunds WHERE external_ref=$1 AND provider_refund_id IS NULL FOR UPDATE`, refundID); err != nil {
			return err
		}
		if len(legacy) > 1 {
			return ErrLedgerConflict
		}
		if len(legacy) == 1 {
			if legacy[0].PaymentID != payment || !legacy[0].Amount.Equal(decimal.NewFromInt(refund.Amount).Shift(-2)) {
				return ErrLedgerConflict
			}
			if _, err = tx.ExecContext(ctx, `UPDATE at_refunds SET provider_refund_id=$2 WHERE id=$1`, legacy[0].ID, refundID); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO at_refunds(payment_id,refund_date,amount,reason,external_ref,provider_refund_id) VALUES($1,CURRENT_TIMESTAMP,$2,'provider-confirmed refund',$3,$3) ON CONFLICT(provider_refund_id) WHERE provider_refund_id IS NOT NULL DO NOTHING`, payment, decimal.NewFromInt(refund.Amount).Shift(-2), refundID); err != nil {
			return err
		}
		if err = Sync(ctx, tx, booking); err != nil {
			return err
		}
	}
	return tx.Commit()
}
