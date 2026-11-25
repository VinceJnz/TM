package stripe

import (
	"api-server/v2/app/services/checkout"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"errors"
	stripe "github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"strconv"
	"strings"
	"time"
)

func checkoutResult(a *stripe.CheckoutSession) checkout.Session {
	var paidAt time.Time
	if a.PaymentIntent != nil && a.PaymentIntent.LatestCharge != nil {
		charge := a.PaymentIntent.LatestCharge
		// A captured card charge supplies a payment timestamp for recovery.
		// Delayed payment methods require the signed success event.
		if charge.Paid && charge.Captured && charge.Created > 0 && charge.PaymentMethodDetails != nil && charge.PaymentMethodDetails.Card != nil {
			paidAt = time.Unix(charge.Created, 0)
		}
	}
	return checkout.Session{PaidAt: paidAt, ID: a.ID, URL: a.URL, AttemptID: a.Metadata["tm_attempt"], State: string(a.Status), Amount: a.AmountTotal, Currency: string(a.Currency), Mode: string(a.Mode), Paid: a.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid}
}
func (g *Gateway) Create(ctx context.Context, a paymentLedger.Attempt) (checkout.Session, error) {
	if g == nil || g.Client == nil {
		return checkout.Session{}, errors.New("Stripe gateway unavailable")
	}
	client := session.Client{B: g.backend, Key: g.apiKey}
	product := &stripe.CheckoutSessionLineItemPriceDataProductDataParams{Name: stripe.String(a.Title)}
	if strings.TrimSpace(a.Description) != "" {
		product.Description = stripe.String(a.Description)
	}
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{{PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{Currency: stripe.String("nzd"), UnitAmount: stripe.Int64(a.Amount), ProductData: product}, Quantity: stripe.Int64(1)}},
		Mode:      stripe.String("payment"), ExpiresAt: stripe.Int64(a.Expires.Unix()), SuccessURL: stripe.String(a.Domain + "/bookings/checkout/success/" + strconv.Itoa(a.BookingID)), CancelURL: stripe.String(a.Domain + "/bookings/checkout/cancel/" + strconv.Itoa(a.BookingID)), InvoiceCreation: &stripe.CheckoutSessionInvoiceCreationParams{Enabled: stripe.Bool(true)}}
	email := a.Email
	if g.RecipientOverride != "" {
		email = g.RecipientOverride
	}
	if email != "" {
		params.CustomerEmail = stripe.String(email)
	}
	params.Context = ctx
	params.SetIdempotencyKey("tm-checkout-" + a.ID)
	params.AddMetadata("tm_attempt", a.ID)
	result, err := client.New(params)
	if err != nil {
		return checkout.Session{}, err
	}
	if g.testMode && result.Livemode {
		return checkout.Session{}, errors.New("live Stripe response rejected in test mode")
	}
	return checkoutResult(result), nil
}
func (g *Gateway) Lookup(ctx context.Context, a paymentLedger.Attempt) ([]checkout.Session, error) {
	if g == nil || g.Client == nil {
		return nil, errors.New("Stripe gateway unavailable")
	}
	client := session.Client{B: g.backend, Key: g.apiKey}
	if a.SessionID.Valid {
		params := &stripe.CheckoutSessionParams{}
		params.AddExpand("payment_intent.latest_charge")
		params.Context = ctx
		result, err := client.Get(a.SessionID.String, params)
		if err != nil {
			return nil, err
		}
		if g.testMode && result.Livemode {
			return nil, errors.New("live Stripe response rejected in test mode")
		}
		return []checkout.Session{checkoutResult(result)}, nil
	}
	params := &stripe.CheckoutSessionListParams{CreatedRange: &stripe.RangeQueryParams{GreaterThanOrEqual: a.Expires.Add(-40 * time.Minute).Unix(), LesserThanOrEqual: a.Expires.Unix()}}
	params.Context = ctx
	params.Limit = stripe.Int64(100)
	params.AddExpand("data.payment_intent.latest_charge")
	results := []checkout.Session{}
	iter := client.List(params)
	count := 0
	for iter.Next() {
		count++
		if count > 1000 {
			return nil, errors.New("checkout reconciliation lookup exceeded bound")
		}
		result := iter.CheckoutSession()
		if result.Metadata["tm_attempt"] == a.ID {
			if g.testMode && result.Livemode {
				return nil, errors.New("live Stripe response rejected in test mode")
			}
			results = append(results, checkoutResult(result))
		}
	}
	return results, iter.Err()
}
