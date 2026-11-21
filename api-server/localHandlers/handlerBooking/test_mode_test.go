package handlerBooking

import (
	"api-server/v2/app/appCore"
	"bytes"
	"fmt"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
	"net/http/httptest"
	"testing"
)

func TestTestModeRejectsSignedLiveEventsAndSessions(t *testing.T) {
	for _, eventLive := range []bool{true, false} {
		app := appCore.New(false)
		app.Settings.StripeTestMode = true
		app.Settings.StripeWebhookSecret = "synthetic-secret"
		payload := []byte(fmt.Sprintf(`{"id":"evt_synthetic","object":"event","api_version":%q,"type":"checkout.session.completed","livemode":%t,"data":{"object":{"object":"checkout.session","id":"cs_synthetic","livemode":true,"status":"complete","payment_status":"paid","mode":"payment","amount_total":1000,"currency":"nzd"}}}`, stripe.APIVersion, eventLive))
		signature := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payload, Secret: app.Settings.StripeWebhookSecret})
		r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payload))
		r.Header.Set("Stripe-Signature", signature.Header)
		w := httptest.NewRecorder()
		New(app).CheckoutWebhook(w, r)
		if w.Code != 400 {
			t.Fatal("live mode reached ledger", w.Code)
		}
	}
}
