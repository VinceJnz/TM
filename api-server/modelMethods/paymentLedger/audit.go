package paymentLedger

import "context"

type auditKey struct{}
type auditIdentity struct{ actor, reason string }

func WithAudit(ctx context.Context, actor, reason string) context.Context {
	return context.WithValue(ctx, auditKey{}, auditIdentity{actor, reason})
}
