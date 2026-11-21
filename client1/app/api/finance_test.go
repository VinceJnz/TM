//go:build js && wasm

package api

import (
	"client1/v2/app/httpProcessor"
	"math"
	"syscall/js"
	"testing"
	"time"
)

func TestFinanceUsesMutableContractAndPropagatesFailure(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `
 const data=JSON.parse(options.body);if('id' in data||'stripe_session_id' in data||'created' in data)throw Error('read-only fields transmitted');
 if(data.booking_id!==11||data.amount!==12.5)throw Error('contract changed');
 return Promise.resolve(new Response('conflict',{status:409}));`))
	finance := NewFinance(httpProcessor.New("https://example.test/custom/v9"))
	done := make(chan error, 1)
	go func() { done <- finance.SavePayment(1, PaymentWrite{BookingID: 11, Amount: 12.5}) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("failed payment reported success")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("finance write blocked")
	}
	if err := finance.DeleteRefund(0); err == nil {
		t.Fatal("invalid ID accepted")
	}
	if err := finance.SavePayment(1, PaymentWrite{BookingID: 11, Amount: math.NaN()}); err == nil {
		t.Fatal("encoding failure reported success")
	}
}
