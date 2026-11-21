package checkout

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"strings"
)

type Refund struct {
	ID, SessionID, Currency, Status string
	Amount                          int64
}
type RefundProvider interface {
	LookupRefund(context.Context, string, string) (Refund, error)
}

// ConfirmRefund observes a refund already requested through the provider's
// approved workflow. It does not issue money or trust client-supplied amounts.
func ConfirmRefund(ctx context.Context, db *sqlx.DB, provider RefundProvider, payment int, refundID, actor string) error {
	if payment <= 0 || !strings.HasPrefix(refundID, "re_") || strings.TrimSpace(actor) == "" {
		return paymentLedger.ErrValidation
	}
	var session string
	if err := db.GetContext(ctx, &session, `SELECT stripe_session_id FROM at_payments WHERE id=$1 AND stripe_session_id IS NOT NULL`, payment); err != nil {
		return err
	}
	// Preserve an observable pending request even if the provider is unavailable.
	result, err := db.ExecContext(ctx, `INSERT INTO at_provider_refunds(id,payment_id,actor,status) VALUES($1,$2,$3,'pending') ON CONFLICT(id) DO UPDATE SET checked_at=CURRENT_TIMESTAMP WHERE at_provider_refunds.payment_id=EXCLUDED.payment_id`, refundID, payment, actor)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return paymentLedger.ErrLedgerConflict
	}
	refund, err := provider.LookupRefund(ctx, session, refundID)
	if err != nil {
		return err
	}
	if refund.ID != refundID || refund.SessionID != session || refund.Currency != "nzd" || refund.Amount <= 0 {
		return paymentLedger.ErrInvalidPayment
	}
	switch refund.Status {
	case "pending", "requires_action", "succeeded", "failed", "canceled":
	default:
		return errors.New("unsupported provider refund status")
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	booking, err := paymentLedger.LockPayment(ctx, tx, payment)
	if err != nil {
		return err
	}
	var oldStatus string
	if err = tx.GetContext(ctx, &oldStatus, `SELECT status FROM at_provider_refunds WHERE id=$1 FOR UPDATE`, refundID); err != nil {
		return err
	}
	if oldStatus == "succeeded" && refund.Status != "succeeded" {
		return paymentLedger.ErrLedgerConflict
	}
	if oldStatus == "succeeded" {
		var oldAmount int64
		if err = tx.GetContext(ctx, &oldAmount, `SELECT amount_minor FROM at_provider_refunds WHERE id=$1`, refundID); err != nil {
			return err
		}
		if oldAmount != refund.Amount {
			return paymentLedger.ErrLedgerConflict
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE at_provider_refunds SET status=$2,amount_minor=$3,checked_at=CURRENT_TIMESTAMP WHERE id=$1`, refundID, refund.Status, refund.Amount); err != nil {
		return err
	}
	if refund.Status == "succeeded" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO at_refunds(payment_id,refund_date,amount,reason,external_ref,provider_refund_id) VALUES($1,CURRENT_TIMESTAMP,$2,'provider-confirmed refund',$3,$3) ON CONFLICT(provider_refund_id) WHERE provider_refund_id IS NOT NULL DO NOTHING`, payment, decimal.NewFromInt(refund.Amount).Shift(-2), refundID); err != nil {
			return err
		}
		if err = paymentLedger.Sync(ctx, tx, booking); err != nil {
			return err
		}
	}
	return tx.Commit()
}
