package bookingPaymentView

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
	blurFunc  js.Func
	focusFunc js.Func
	cleanedUp bool
}

// MakePayment initiates the payment session
func (p *ItemEditor) MakePayment(ItemID int64) {
	var checkoutResp CheckoutCreateResponse
	log.Printf("%vMakePayment() starting for BookingID=%d", debugTag, ItemID)

	success := func(err error) {
		if err != nil {
			log.Printf("%vMakePayment()1 error: %v", debugTag, err)
			//p.updateStateDisplay(viewHelpers.ItemStateNone)
			return
		}

		// Extract payment URL from response and Validate checkout URL
		if checkoutResp.CheckoutURL == "" {
			log.Printf("%vMakePayment()2 no checkout URL received", debugTag)
			//p.updateStateDisplay(viewHelpers.ItemStateNone)
			return
		}

		log.Printf("%vMakePayment()3 received checkout URL: %s", debugTag, checkoutResp.CheckoutURL)

		//p.populateItemList()
		//p.updateStateDisplay(viewHelpers.ItemStateNone)
		p.openPaymentPage(checkoutResp.CheckoutURL)
	}

	fail := func(err error) {
		log.Printf("%vMakePayment()4 failed: %v", debugTag, err)
		//p.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	// POST request to create checkout session
	p.client.NewRequest(
		http.MethodPost,
		ApiURL+"/checkout/create/"+strconv.FormatInt(ItemID, 10),
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
	if !p.UiComponents.paymentWindow.IsNull() && !p.UiComponents.paymentWindow.IsUndefined() {
		log.Println(debugTag+"windowBlur() payment window closed =",
			p.UiComponents.paymentWindow.Get("closed").Bool())
	}
	return nil
}

// windowFocus triggered when focus is received
func (p *ItemEditor) windowFocus(this js.Value, args []js.Value) interface{} {
	if p.UiComponents.paymentWindow.IsNull() || p.UiComponents.paymentWindow.IsUndefined() {
		return nil
	}

	if p.UiComponents.paymentWindow.Get("closed").Bool() {
		p.paymentWindowDestroy()
		return nil
	}

	p.checkPayment()
	return nil
}

// paymentWindowCreate creates the new payment window and sets up event listeners
func (p *ItemEditor) paymentWindowCreate(url string) {
	global := js.Global()
	window := global.Get("window")

	// Create cleanup functions that can be removed later
	blurFunc := js.FuncOf(p.windowBlur)
	focusFunc := js.FuncOf(p.windowFocus)

	// Store cleanup functions
	if p.UiComponents.eventCleanup == nil {
		p.UiComponents.eventCleanup = &eventCleanup{}
	}
	p.UiComponents.eventCleanup.blurFunc = blurFunc
	p.UiComponents.eventCleanup.focusFunc = focusFunc

	window.Call("addEventListener", "blur", blurFunc)
	window.Call("addEventListener", "focus", focusFunc)

	p.UiComponents.paymentWindow = window.Call("open", url, "_blank")

	if p.UiComponents.paymentWindow.IsNull() || p.UiComponents.paymentWindow.IsUndefined() {
		log.Printf("%vpaymentWindowCreate() failed to open payment window - popup may be blocked", debugTag)
		p.cleanupEventListeners()
	} else {
		log.Printf("%vpaymentWindowCreate() payment window opened successfully", debugTag)
	}
}

// cleanupEventListeners removes event listeners and releases js.Func
func (p *ItemEditor) cleanupEventListeners() {
	log.Printf("%vcleanupEventListeners() called, eventCleanup: %T, %+v", debugTag, p.UiComponents.eventCleanup, p.UiComponents.eventCleanup)
	if p.UiComponents.eventCleanup == nil {
		return
	}

	// Prevent double cleanup
	if p.UiComponents.eventCleanup.cleanedUp {
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
		window.Call("removeEventListener", "blur", p.UiComponents.eventCleanup.blurFunc)
		p.UiComponents.eventCleanup.blurFunc.Release()
		log.Printf("%vcleanupEventListeners() blurFunc cleaned up", debugTag)
	}()

	// Try to remove focus listener
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("%vcleanupEventListeners() failed to cleanup focusFunc: %v", debugTag, r)
			}
		}()
		window.Call("removeEventListener", "focus", p.UiComponents.eventCleanup.focusFunc)
		p.UiComponents.eventCleanup.focusFunc.Release()
		log.Printf("%vcleanupEventListeners() focusFunc cleaned up", debugTag)
	}()

	// Mark as cleaned up
	p.UiComponents.eventCleanup.cleanedUp = true

	log.Printf("%vcleanupEventListeners() cleanup completed", debugTag)

	//p.UiComponents.eventCleanup = nil
}

// paymentWindowDestroy handles cleanup after payment window closes
func (p *ItemEditor) paymentWindowDestroy() {
	// Clean up event listeners first
	p.cleanupEventListeners()

	// Note: The server doesn't have a /closed endpoint
	// Instead, we should call /check to get the final status
	// Or rely on the /success and /cancel callbacks from Stripe

	/*
		success := func(err error, data *httpProcessor.ReturnData) {
			log.Printf("%vpaymentWindowDestroy() success", debugTag)
			// Refresh the booking list to show updated payment status
			p.RecordState = RecordStateReloadRequired
			//p.FetchItems() //?????????
		}

		fail := func(err error, data *httpProcessor.ReturnData) {
			log.Printf("%vpaymentWindowDestroy() error: %v", debugTag, err)
		}

		// Check final payment status
		p.client.NewRequest(
			http.MethodGet,
			ApiURL+"/checkout/close/"+strconv.FormatInt(int64(p.ParentData.ID), 10),
			nil,
			nil,
			success,
			fail,
		)
	*/
}

// checkPayment triggers payment status check
func (p *ItemEditor) checkPayment() {
	var statusResp CheckoutStatusResponse

	log.Printf("%vcheckPayment() checking payment status for BookingID=%d", debugTag, p.ParentData.ID)

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
			p.UiComponents.paymentWindow.Call("close")
			p.paymentWindowDestroy()
			// Refresh booking list
			p.RecordState = RecordStateReloadRequired

		case "expired": // stripe.CheckoutSessionStatusExpired: //"expired":
			// Payment expired or was canceled
			log.Printf("%vcheckPayment() payment %s", debugTag, statusResp.Status)
			p.UiComponents.paymentWindow.Call("close")
			p.paymentWindowDestroy()

		default:
			log.Printf("%vcheckPayment() unknown status: %s", debugTag, statusResp.Status)
		}
	}

	fail := func(err error) {
		log.Printf("%vcheckPayment() failed: %v", debugTag, err)
	}

	log.Printf("%vcheckPayment() checking payment status for BookingID=%d", debugTag, p.ParentData.ID)

	p.client.NewRequest(
		http.MethodGet,
		ApiURL+"/checkout/check/"+strconv.FormatInt(int64(p.ParentData.ID), 10),
		&statusResp,
		nil,
		success,
		fail,
	)
}
