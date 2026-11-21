// Provider orchestration is separate from SQL observations and ledger commands.
package checkout

import (
	"api-server/v2/modelMethods/paymentLedger"
	"api-server/v2/modelMethods/refunds"
	"context"
	"github.com/jmoiron/sqlx"
	"strings"
)

type Refund = paymentLedger.ProviderRefund
type RefundProvider interface {
	LookupRefund(context.Context, string, string) (Refund, error)
}

func ConfirmRefund(ctx context.Context, db *sqlx.DB, provider RefundProvider, payment int, refundID, actor string) error {
	if payment <= 0 || !strings.HasPrefix(refundID, "re_") || strings.TrimSpace(actor) == "" {
		return paymentLedger.ErrValidation
	}
	store := refunds.Store{DB: db}
	session, err := store.PaymentSession(ctx, payment)
	if err != nil {
		return err
	}
	if err = store.Observe(ctx, payment, refundID, actor); err != nil {
		return err
	}
	result, err := provider.LookupRefund(ctx, session, refundID)
	if err != nil {
		return err
	}
	if result.ID != refundID || result.SessionID != session {
		return paymentLedger.ErrInvalidPayment
	}
	return paymentLedger.ApplyProviderRefund(ctx, db, payment, actor, result)
}
