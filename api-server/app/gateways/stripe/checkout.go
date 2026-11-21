package stripe

import (
	"api-server/v2/app/services/checkout"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"errors"
	stripe "github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"strconv"
	"time"
)

func checkoutResult(a *stripe.CheckoutSession) checkout.Session {
	return checkout.Session{ID: a.ID, URL: a.URL, AttemptID: a.Metadata["tm_attempt"], State: string(a.Status), Amount: a.AmountTotal, Currency: string(a.Currency), Mode: string(a.Mode), Paid: a.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid}
}
func (g *Gateway) Create(ctx context.Context, a paymentLedger.Attempt) (checkout.Session, error) {
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{{PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{Currency: stripe.String("nzd"), UnitAmount: stripe.Int64(a.Amount), ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{Name: stripe.String(a.Title), Description: stripe.String(a.Description)}}, Quantity: stripe.Int64(1)}},
		Mode:      stripe.String("payment"), ExpiresAt: stripe.Int64(a.Expires.Unix()), SuccessURL: stripe.String(a.Domain + "/bookings/checkout/success/" + strconv.Itoa(a.BookingID)), CancelURL: stripe.String(a.Domain + "/bookings/checkout/cancel/" + strconv.Itoa(a.BookingID)), InvoiceCreation: &stripe.CheckoutSessionInvoiceCreationParams{Enabled: stripe.Bool(true)}}
	if a.Email != "" {
		params.CustomerEmail = stripe.String(a.Email)
	}
	params.Context = ctx
	params.SetIdempotencyKey("tm-checkout-" + a.ID)
	params.AddMetadata("tm_attempt", a.ID)
	result, err := session.New(params)
	if err != nil {
		return checkout.Session{}, err
	}
	return checkoutResult(result), nil
}
func (g *Gateway) Lookup(ctx context.Context, a paymentLedger.Attempt) ([]checkout.Session, error) {
	if a.SessionID.Valid {
		params := &stripe.CheckoutSessionParams{}
		params.Context = ctx
		result, err := session.Get(a.SessionID.String, params)
		if err != nil {
			return nil, err
		}
		return []checkout.Session{checkoutResult(result)}, nil
	}
	params := &stripe.CheckoutSessionListParams{CreatedRange: &stripe.RangeQueryParams{GreaterThanOrEqual: a.Expires.Add(-40 * time.Minute).Unix(), LesserThanOrEqual: a.Expires.Unix()}}
	params.Context = ctx
	params.Limit = stripe.Int64(100)
	results := []checkout.Session{}
	iter := session.List(params)
	count := 0
	for iter.Next() {
		count++
		if count > 1000 {
			return nil, errors.New("checkout reconciliation lookup exceeded bound")
		}
		result := iter.CheckoutSession()
		if result.Metadata["tm_attempt"] == a.ID {
			results = append(results, checkoutResult(result))
		}
	}
	return results, iter.Err()
}
