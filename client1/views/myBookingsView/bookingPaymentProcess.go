package myBookingsView

import (
	"log"
	"net/http"
	"strconv"
	"syscall/js"
)

// PaymentResponse represents the response from the checkout create endpoint
type CheckoutCreateResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
	Status      string `json:"status"`
}

// PaymentCheckResponse represents the response from the checkout check endpoint
type CheckoutStatusResponse struct {
	Status      string  `json:"status"`
	AmountTotal float64 `json:"amount_total,omitempty"`
}

//*************************************************************
// Manage the opening and closing of the payment gateway page
//*************************************************************
// Sequence:
// 1. User clicks "Make Payment" button -> MakePayment()
// 2. MakePayment() calls API to create checkout session -> receives checkout URL
// 3. openPaymentPage() opens new browser tab with payment URL
// 4. Event listeners monitor window focus/blur
// 5. On focus, checkPayment() checks payment status via API
// 6. If payment complete/canceled/expired, close payment window and cleanup
//*************************************************************

// Store event listener cleanup functions
type eventCleanup struct {
	blurFunc    js.Func
	focusFunc   js.Func
	messageFunc js.Func
	cleanedUp   bool
}

// MakePayment initiates the payment session
func (p *ItemEditor) MakePayment() {
	var checkoutResp CheckoutCreateResponse
	log.Printf("%vMakePayment() starting for BookingID=%d", debugTag, p.CurrentRecord.ID)

	success := func(err error) {
		if err != nil {
			log.Printf("%vMakePayment()1 error: %v", debugTag, err)
			return
		}

		// Extract and validate checkout URL.
		if checkoutResp.CheckoutURL == "" {
			log.Printf("%vMakePayment()2 no checkout URL received", debugTag)
			return
		}

		log.Printf("%vMakePayment()3 received checkout URL: %s", debugTag, checkoutResp.CheckoutURL)
		p.openPaymentPage(checkoutResp.CheckoutURL)
	}

	fail := func(err error) {
		log.Printf("%vMakePayment()4 failed: %v", debugTag, err)
	}

	// POST request to create checkout session
	p.client.NewRequest(
		http.MethodPost,
		ApiURLBookings+"/checkout/create/"+strconv.FormatInt(int64(p.CurrentRecord.ID), 10),
		&checkoutResp,
		nil,
		success,
		fail,
	)
}

// openPaymentPage opens a new browser tab with the payment page URL
func (p *ItemEditor) openPaymentPage(paymentURL string) {
	if paymentURL == "" {
		log.Printf("%vopenPaymentPage() called with empty URL", debugTag)
		return
	}
	p.paymentWindowCreate(paymentURL)
}

// windowBlur triggered when window loses focus
func (p *ItemEditor) windowBlur(this js.Value, args []js.Value) interface{} {
	if !p.Children.PaymentState.paymentWindow.IsNull() && !p.Children.PaymentState.paymentWindow.IsUndefined() {
		log.Println(debugTag+"windowBlur() payment window closed =",
			p.Children.PaymentState.paymentWindow.Get("closed").Bool())
	}
	return nil
}

// windowFocus triggered when focus is received
func (p *ItemEditor) windowFocus(this js.Value, args []js.Value) interface{} {
	if p.Children.PaymentState.paymentWindow.IsNull() || p.Children.PaymentState.paymentWindow.IsUndefined() {
		return nil
	}

	if p.Children.PaymentState.paymentWindow.Get("closed").Bool() {
		p.paymentWindowDestroy()
		return nil
	}

	p.checkPayment()
	return nil
}

// windowMessage handles postMessage callbacks from the payment popup.
func (p *ItemEditor) windowMessage(this js.Value, args []js.Value) interface{} {
	if len(args) == 0 {
		return nil
	}

	event := args[0]
	data := event.Get("data")
	if data.IsUndefined() || data.IsNull() {
		return nil
	}

	msgType := data.Get("type")
	if msgType.IsUndefined() || msgType.String() != "tm-payment-status" {
		return nil
	}

	bookingID := data.Get("bookingId")
	if !bookingID.IsUndefined() && !bookingID.IsNull() && bookingID.Int() != p.CurrentRecord.ID {
		return nil
	}

	status := data.Get("status").String()
	log.Printf("%vwindowMessage() received payment status=%s", debugTag, status)

	if !p.Children.PaymentState.paymentWindow.IsNull() && !p.Children.PaymentState.paymentWindow.IsUndefined() {
		p.Children.PaymentState.paymentWindow.Call("close")
	}
	p.paymentWindowDestroy()

	if status == "complete" {
		p.RecordState = RecordStateReloadRequired
		p.FetchItems()
	}

	return nil
}

// paymentWindowCreate creates the new payment window and sets up event listeners
func (p *ItemEditor) paymentWindowCreate(url string) {
	global := js.Global()
	window := global.Get("window")

	// Create cleanup functions that can be removed later
	blurFunc := js.FuncOf(p.windowBlur)
	focusFunc := js.FuncOf(p.windowFocus)
	messageFunc := js.FuncOf(p.windowMessage)

	// Store cleanup functions
	if p.Children.PaymentState.eventCleanup == nil {
		p.Children.PaymentState.eventCleanup = &eventCleanup{}
	}
	p.Children.PaymentState.eventCleanup.cleanedUp = false
	p.Children.PaymentState.eventCleanup.blurFunc = blurFunc
	p.Children.PaymentState.eventCleanup.focusFunc = focusFunc
	p.Children.PaymentState.eventCleanup.messageFunc = messageFunc
	window.Call("addEventListener", "blur", blurFunc)
	window.Call("addEventListener", "focus", focusFunc)
	window.Call("addEventListener", "message", messageFunc)

	p.Children.PaymentState.paymentWindow = window.Call("open", url, "_blank")
	if p.Children.PaymentState.paymentWindow.IsNull() || p.Children.PaymentState.paymentWindow.IsUndefined() {
		log.Printf("%vpaymentWindowCreate() failed to open payment window - popup may be blocked", debugTag)
		p.cleanupEventListeners()
	} else {
		log.Printf("%vpaymentWindowCreate() payment window opened successfully", debugTag)
	}
}

// cleanupEventListeners removes event listeners and releases js.Func
func (p *ItemEditor) cleanupEventListeners() {
	log.Printf("%vcleanupEventListeners() called, eventCleanup: %T, %+v", debugTag, p.Children.PaymentState.eventCleanup, p.Children.PaymentState.eventCleanup)
	if p.Children.PaymentState.eventCleanup == nil {
		return
	}

	// Prevent double cleanup
	if p.Children.PaymentState.eventCleanup.cleanedUp {
		log.Printf("%vcleanupEventListeners() already cleaned up, returning", debugTag)
		return
	}

	global := js.Global()
	window := global.Get("window")

	// Only remove and release if the js.Func was actually created
	// Check by seeing if the Value has a valid reference
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%vcleanupEventListeners() recovered from panic: %v", debugTag, r)
		}
	}()

	// Try to remove blur listener - wrap in individual defer to continue if one fails
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("%vcleanupEventListeners() failed to cleanup blurFunc: %v", debugTag, r)
			}
		}()
		window.Call("removeEventListener", "blur", p.Children.PaymentState.eventCleanup.blurFunc)
		p.Children.PaymentState.eventCleanup.blurFunc.Release()
		log.Printf("%vcleanupEventListeners() blurFunc cleaned up", debugTag)
	}()

	// Try to remove focus listener
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("%vcleanupEventListeners() failed to cleanup focusFunc: %v", debugTag, r)
			}
		}()
		window.Call("removeEventListener", "focus", p.Children.PaymentState.eventCleanup.focusFunc)
		p.Children.PaymentState.eventCleanup.focusFunc.Release()
		log.Printf("%vcleanupEventListeners() focusFunc cleaned up", debugTag)
	}()

	// Try to remove message listener
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("%vcleanupEventListeners() failed to cleanup messageFunc: %v", debugTag, r)
			}
		}()
		window.Call("removeEventListener", "message", p.Children.PaymentState.eventCleanup.messageFunc)
		p.Children.PaymentState.eventCleanup.messageFunc.Release()
		log.Printf("%vcleanupEventListeners() messageFunc cleaned up", debugTag)
	}()

	// Mark as cleaned up
	p.Children.PaymentState.eventCleanup.cleanedUp = true

	log.Printf("%vcleanupEventListeners() cleanup completed", debugTag)
}

// paymentWindowDestroy handles cleanup after payment window closes
func (p *ItemEditor) paymentWindowDestroy() {
	// Clean up event listeners first
	p.cleanupEventListeners()

	// Note: The server doesn't have a /closed endpoint
	// Instead, we should call /check to get the final status
	// Or rely on the /success and /cancel callbacks from Stripe
}

// checkPayment triggers payment status check
func (p *ItemEditor) checkPayment() {
	var statusResp CheckoutStatusResponse

	log.Printf("%vcheckPayment() checking payment status for BookingID=%d", debugTag, p.CurrentRecord.ID)

	success := func(err error) {
		if err != nil {
			log.Printf("%vcheckPayment() error: %v", debugTag, err)
			return
		}

		log.Printf("%vcheckPayment() status: %s", debugTag, statusResp.Status)

		switch statusResp.Status {
		case "open": // stripe.CheckoutSessionStatusOpen: //"open":
			// Payment still in progress, do nothing
			log.Printf("%vcheckPayment() payment still open", debugTag)

		case "complete": // "complete":
			// Payment completed successfully
			p.Children.PaymentState.paymentWindow.Call("close")
			p.paymentWindowDestroy()
			// Refresh booking list
			p.RecordState = RecordStateReloadRequired
			p.FetchItems()

		case "expired": // stripe.CheckoutSessionStatusExpired: //"expired":
			// Payment expired or was canceled
			log.Printf("%vcheckPayment() payment %s", debugTag, statusResp.Status)
			p.Children.PaymentState.paymentWindow.Call("close")
			p.paymentWindowDestroy()

		default:
			log.Printf("%vcheckPayment() unknown status: %s", debugTag, statusResp.Status)
		}
	}

	fail := func(err error) {
		log.Printf("%vcheckPayment() failed: %v", debugTag, err)
	}

	p.client.NewRequest(
		http.MethodGet,
		ApiURLBookings+"/checkout/check/"+strconv.FormatInt(int64(p.CurrentRecord.ID), 10),
		&statusResp,
		nil,
		success,
		fail,
	)
}
