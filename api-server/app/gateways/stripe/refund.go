package stripe

import (
	"api-server/v2/app/services/checkout"
	"context"
	"errors"
	stripe "github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/refund"
)

func (g *Gateway) LookupRefund(ctx context.Context, sessionID, refundID string) (checkout.Refund, error) {
	if g == nil || g.Client == nil {
		return checkout.Refund{}, errors.New("Stripe gateway unavailable")
	}
	sessions := session.Client{B: g.backend, Key: g.apiKey}
	refunds := refund.Client{B: g.backend, Key: g.apiKey}
	sp := &stripe.CheckoutSessionParams{}
	sp.Context = ctx
	s, err := sessions.Get(sessionID, sp)
	if err != nil {
		return checkout.Refund{}, err
	}
	rp := &stripe.RefundParams{}
	rp.Context = ctx
	r, err := refunds.Get(refundID, rp)
	if err != nil {
		return checkout.Refund{}, err
	}
	if g.testMode && s.Livemode {
		return checkout.Refund{}, errors.New("live Stripe response rejected in test mode")
	}
	if s.PaymentIntent == nil || r.PaymentIntent == nil || s.PaymentIntent.ID != r.PaymentIntent.ID {
		return checkout.Refund{}, errors.New("refund belongs to a different payment")
	}
	return checkout.Refund{ID: r.ID, SessionID: sessionID, Currency: string(r.Currency), Status: string(r.Status), Amount: r.Amount}, nil
}
