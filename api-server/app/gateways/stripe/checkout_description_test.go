package stripe

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdk "github.com/stripe/stripe-go/v84"
)

func TestCheckoutSendsBookedDatesAndOmitsLegacyBlankDescription(t *testing.T) {
	const field = "line_items[0][price_data][product_data][description]"
	requests := make(chan map[string][]string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		requests <- r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"cs_synthetic","object":"checkout.session"}`))
	}))
	defer server.Close()

	old := sdk.GetBackend(sdk.APIBackend)
	oldKey := sdk.Key
	defer func() { sdk.SetBackend(sdk.APIBackend, old); sdk.Key = oldKey }()
	sdk.SetBackend(sdk.APIBackend, sdk.GetBackendWithConfig(sdk.APIBackend, &sdk.BackendConfig{URL: sdk.String(server.URL), HTTPClient: server.Client(), MaxNetworkRetries: sdk.Int64(0)}))
	gateway := NewFromKey("sk_test_synthetic-checkout", "", true)
	for _, value := range []string{"Booking dates: 2026-01-02 to 2026-01-04", "  "} {
		attempt := paymentLedger.Attempt{ID: "attempt", BookingID: 10, Amount: 500, Title: "Example trip", Description: value, Domain: "https://example.test", Expires: time.Now().Add(time.Hour)}
		if _, err := gateway.Create(context.Background(), attempt); err != nil {
			t.Fatal(err)
		}
		fields := <-requests
		if value == "  " {
			if _, sent := fields[field]; sent {
				t.Fatal("blank description was sent")
			}
		} else if description := fields[field]; len(description) != 1 || description[0] != value {
			t.Fatalf("unexpected checkout description: %q", description)
		}
	}
}
