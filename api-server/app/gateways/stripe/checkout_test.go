package stripe

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"database/sql"
	"encoding/json"
	sdk "github.com/stripe/stripe-go/v84"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckoutProviderContractAndCancellation(t *testing.T) {
	attempt := paymentLedger.Attempt{ID: "synthetic-attempt", BookingID: 1, Amount: 1000, Email: "member@example.test", Title: "Synthetic trip", Domain: "https://app.example.test/custom/v9", Expires: time.Now().Add(time.Hour)}
	var requests atomic.Int32
	var fail atomic.Bool
	result := map[string]any{"object": "checkout.session", "id": "cs_synthetic", "url": "https://checkout.example.test", "metadata": map[string]string{"tm_attempt": attempt.ID}, "amount_total": 1000, "currency": "nzd", "mode": "payment", "status": "open", "payment_status": "unpaid"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_synthetic-contract" {
			t.Error("gateway used another instance's key")
		}
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if fail.Load() {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":{"type":"api_error","message":"synthetic failure"}}`))
			return
		}
		if r.Method == "POST" {
			r.ParseForm()
			if r.Form.Get("customer_email") != "tester@example.test" {
				t.Error("registered address escaped test routing")
			}
			if r.Header.Get("Idempotency-Key") != "tm-checkout-"+attempt.ID || r.Form.Get("line_items[0][price_data][unit_amount]") != "1000" || r.Form.Get("metadata[tm_attempt]") != attempt.ID {
				t.Error("provider quote/idempotency contract changed")
			}
		}
		if r.Method == "GET" && r.URL.Path == "/v1/checkout/sessions" {
			if r.URL.Query().Get("created[gte]") == "" {
				t.Error("recovery lookup is unbounded")
			}
			json.NewEncoder(w).Encode(map[string]any{"object": "list", "has_more": false, "data": []any{result}})
			return
		}
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()
	oldBackend := sdk.GetBackend(sdk.APIBackend)
	oldKey := sdk.Key
	defer func() { sdk.SetBackend(sdk.APIBackend, oldBackend); sdk.Key = oldKey }()
	sdk.SetBackend(sdk.APIBackend, sdk.GetBackendWithConfig(sdk.APIBackend, &sdk.BackendConfig{URL: sdk.String(server.URL), HTTPClient: server.Client(), MaxNetworkRetries: sdk.Int64(0)}))
	gateway := NewFromKey("sk_test_synthetic-contract", attempt.Domain, true)
	gateway.RecipientOverride = "tester@example.test"
	sdk.Key = "sk_live_synthetic-other-instance"
	for i := 0; i < 2; i++ {
		created, err := gateway.Create(context.Background(), attempt)
		if err != nil || created.ID != "cs_synthetic" {
			t.Fatal(created, err)
		}
	}
	results, err := gateway.Lookup(context.Background(), attempt)
	if err != nil || len(results) != 1 {
		t.Fatal(results, err)
	}
	attempt.SessionID = sql.NullString{String: "cs_synthetic", Valid: true}
	results, err = gateway.Lookup(context.Background(), attempt)
	if err != nil || len(results) != 1 {
		t.Fatal(results, err)
	}
	before := requests.Load()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = gateway.Create(ctx, attempt); err == nil || requests.Load() != before {
		t.Fatal("cancelled request reached provider", err)
	}
	result["livemode"] = true
	if _, err = gateway.Create(context.Background(), attempt); err == nil {
		t.Fatal("live response accepted in test mode")
	}
	result["livemode"] = false
	blocked := NewFromKey("sk_live_synthetic", attempt.Domain, true)
	before = requests.Load()
	if _, err = blocked.Create(context.Background(), attempt); err == nil || requests.Load() != before {
		t.Fatal("test mode used a live key")
	}
	fail.Store(true)
	if _, err = gateway.Create(context.Background(), attempt); err == nil {
		t.Fatal("provider failure was swallowed")
	}
}
