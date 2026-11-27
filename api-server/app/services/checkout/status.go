package checkout

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"errors"
	"time"
)

var ErrNoSession = errors.New("booking has no Stripe session")

type Status struct {
	State     string
	Amount    int64
	CheckedAt time.Time
}

// Check reports provider status without changing booking or ledger state.
func (s *Service) Check(ctx context.Context, id, owner int) (Status, error) {
	sessionID, err := paymentLedger.BookingCheckoutSession(ctx, s.DB, id, owner)
	if err != nil {
		return Status{}, err
	}
	if !sessionID.Valid || sessionID.String == "" {
		return Status{}, ErrNoSession
	}
	results, err := s.Provider.Lookup(ctx, paymentLedger.Attempt{SessionID: sessionID})
	if err != nil {
		return Status{}, err
	}
	if len(results) != 1 || results[0].ID != sessionID.String {
		return Status{}, paymentLedger.ErrInvalidPayment
	}
	return Status{State: results[0].State, Amount: results[0].Amount, CheckedAt: s.Now()}, nil
}
