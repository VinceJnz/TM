// Package finance owns authenticated manual financial commands. The ledger
// owns their SQL, locking and transaction boundaries; triggers retain history.
package finance

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
)

type Service struct{ DB *sqlx.DB }

func auditContext(ctx context.Context, actor int, operation, reason string) (context.Context, error) {
	reason = strings.TrimSpace(reason)
	if operation == "create" && reason == "" {
		reason = "manual financial entry"
	}
	if actor <= 0 || reason == "" || len(reason) > 500 {
		return ctx, paymentLedger.ErrValidation
	}
	return paymentLedger.WithAudit(ctx, fmt.Sprintf("user:%d", actor), reason), nil
}
func (s Service) Payment(ctx context.Context, actor int, reason, operation string, id int, input paymentLedger.PaymentWrite) (int, error) {
	ctx, err := auditContext(ctx, actor, operation, reason)
	if err != nil {
		return 0, err
	}
	return paymentLedger.WritePayment(ctx, s.DB, operation, id, input)
}
func (s Service) Refund(ctx context.Context, actor int, reason, operation string, id int, input paymentLedger.RefundWrite) (int, error) {
	ctx, err := auditContext(ctx, actor, operation, reason)
	if err != nil {
		return 0, err
	}
	return paymentLedger.WriteRefund(ctx, s.DB, operation, id, input)
}
