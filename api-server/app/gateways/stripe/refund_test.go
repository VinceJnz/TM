package stripe

import (
	"context"
	"encoding/json"
	sdk "github.com/stripe/stripe-go/v84"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRefundLookupBindsProviderPaymentIntent(t *testing.T) {
	intent := "pi_synthetic"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/checkout/sessions/cs_synthetic" {
			json.NewEncoder(w).Encode(map[string]any{"id": "cs_synthetic", "object": "checkout.session", "payment_intent": "pi_synthetic"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "re_synthetic", "object": "refund", "payment_intent": intent, "amount": 500, "currency": "nzd", "status": "succeeded"})
	}))
	defer server.Close()
	old := sdk.GetBackend(sdk.APIBackend)
	oldKey := sdk.Key
	defer func() { sdk.SetBackend(sdk.APIBackend, old); sdk.Key = oldKey }()
	sdk.SetBackend(sdk.APIBackend, sdk.GetBackendWithConfig(sdk.APIBackend, &sdk.BackendConfig{URL: sdk.String(server.URL), HTTPClient: server.Client(), MaxNetworkRetries: sdk.Int64(0)}))
	g := NewFromKey("sk_test_synthetic-refund", "", true)
	if result, err := g.LookupRefund(context.Background(), "cs_synthetic", "re_synthetic"); err != nil || result.Amount != 500 {
		t.Fatal(result, err)
	}
	intent = "pi_other"
	if _, err := g.LookupRefund(context.Background(), "cs_synthetic", "re_synthetic"); err == nil {
		t.Fatal("accepted another payment's refund")
	}
}
