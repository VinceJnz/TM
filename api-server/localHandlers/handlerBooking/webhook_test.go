package handlerBooking

import (
	"api-server/v2/app/appCore"
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

func TestWebhookRejectsUnsignedAndUnpaidEvents(t *testing.T) {
	app := appCore.New(false)
	app.Settings.StripeWebhookSecret = "synthetic-webhook-secret"
	h := New(app)
	for _, signed := range []bool{false, true} {
		payload := []byte(fmt.Sprintf(`{"id":"synthetic-event","object":"event","api_version":%q,"type":"checkout.session.completed","data":{"object":{"id":"checkout","object":"checkout.session","payment_status":"unpaid","status":"complete","mode":"payment","amount_total":1000,"currency":"nzd"}}}`, stripe.APIVersion))
		r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payload))
		if signed {
			signature := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payload, Secret: app.Settings.StripeWebhookSecret})
			r.Header.Set("Stripe-Signature", signature.Header)
		}
		w := httptest.NewRecorder()
		h.CheckoutWebhook(w, r)
		if w.Code != 400 {
			t.Fatalf("signed=%v: expected 400, got %d", signed, w.Code)
		}
	}
}
