package checkout

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrUnpaidWebhook   = errors.New("checkout is not paid")
	ErrWebhookCurrency = errors.New("invalid currency")
	ErrWebhookRecovery = errors.New("checkout recovery failed")
)

// FinalizeWebhook handles a checkout session from a verified provider event.
// Attaching is retry-safe if finalization fails and Stripe redelivers the event.
func (s *Service) FinalizeWebhook(ctx context.Context, eventID string, session Session, paidAt time.Time) (bool, error) {
	if !session.Paid || session.State != "complete" || session.Mode != "payment" {
		return false, ErrUnpaidWebhook
	}
	if session.AttemptID != "" {
		if session.Currency != "nzd" {
			return false, ErrWebhookCurrency
		}
		if err := paymentLedger.Attach(ctx, s.DB, session.AttemptID, session.ID, "", session.Amount); err != nil {
			return false, fmt.Errorf("%w: %w", ErrWebhookRecovery, err)
		}
	}
	return paymentLedger.Finalize(ctx, s.DB, eventID, session.ID, session.Amount, session.Currency, paidAt)
}
