package myBookingsView

import (
	"client1/v2/app/httpProcessor"
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
func (p *ItemEditor) MakePayment() {
	var checkoutResp CheckoutCreateResponse
	log.Printf("%vMakePayment() starting for BookingID=%d", debugTag, p.CurrentRecord.ID)

	success := func(err error, data *httpProcessor.ReturnData) {
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

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%vMakePayment()4 failed: %v", debugTag, err)
		//p.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	// POST request to create checkout session
	p.client.NewRequest(
		http.MethodPost,
		ApiURL1+"/checkout/create/"+strconv.FormatInt(int64(p.CurrentRecord.ID), 10),
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

// paymentWindowCreate creates the new payment window and sets up event listeners
func (p *ItemEditor) paymentWindowCreate(url string) {
	global := js.Global()
	window := global.Get("window")

	// Create cleanup functions that can be removed later
	blurFunc := js.FuncOf(p.windowBlur)
	focusFunc := js.FuncOf(p.windowFocus)

	// Store cleanup functions
	if p.Children.PaymentState.eventCleanup == nil {
		p.Children.PaymentState.eventCleanup = &eventCleanup{}
	}
	p.Children.PaymentState.eventCleanup.blurFunc = blurFunc
	p.Children.PaymentState.eventCleanup.focusFunc = focusFunc
	window.Call("addEventListener", "blur", blurFunc)
	window.Call("addEventListener", "focus", focusFunc)

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

	// Mark as cleaned up
	p.Children.PaymentState.eventCleanup.cleanedUp = true

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

	log.Printf("%vcheckPayment() checking payment status for BookingID=%d", debugTag, p.CurrentRecord.ID)

	success := func(err error, data *httpProcessor.ReturnData) {
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

		case "expired": // stripe.CheckoutSessionStatusExpired: //"expired":
			// Payment expired or was canceled
			log.Printf("%vcheckPayment() payment %s", debugTag, statusResp.Status)
			p.Children.PaymentState.paymentWindow.Call("close")
			p.paymentWindowDestroy()

		default:
			log.Printf("%vcheckPayment() unknown status: %s", debugTag, statusResp.Status)
		}
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%vcheckPayment() failed: %v", debugTag, err)
	}

	log.Printf("%vcheckPayment() checking payment status for BookingID=%d", debugTag, p.CurrentRecord.ID)

	p.client.NewRequest(
		http.MethodGet,
		ApiURL1+"/checkout/check/"+strconv.FormatInt(int64(p.CurrentRecord.ID), 10),
		&statusResp,
		nil,
		success,
		fail,
	)
}

/*

//*************************************************************
// Manage the opening and closing of the pament gateway page
//*************************************************************

// paymentOpen opens a new browser tab with the payment page url provided from the server
// Once the payment is complete the payment page is directed back to the server to advise it is complete (success or cancelled)
// it continiously checks for closure of the payment page and calls the windowClose if it has been closed
func (p *ItemEditor) openPaymentPage(paymentURL string, err error) {
	p.paymentWindowCreate(paymentURL)
	//p.Callback()
}

func (p *ItemEditor) windowBlur(this js.Value, args []js.Value) interface{} {
	log.Println(debugTag+"ListView.windowBlur()1", "p.paymentWindow.Get(closed) =", p.UiComponents.paymentWindow.Get("closed").Bool())
	return nil
}

// windowFocus triggered when focus is received
// This is used to trigger a check on the payment window to see if the payment is complete
// 1. Check if the payment window is still open
// 2. Check the payment record on the payment site to see if it is complete/cancelled
func (p *ItemEditor) windowFocus(this js.Value, args []js.Value) interface{} {
	if p.UiComponents.paymentWindow.Get("closed").Bool() {
		p.paymentWindowDestroy()
		return nil
	}
	p.checkPayment()
	return nil
}

// createWindow creates the new payment page and directs it to the payment url
// it sets up event listeners on the current page to determine when focus has been returned
func (p *ItemEditor) paymentWindowCreate(url string) {
	global := js.Global()
	window := global.Get("window")
	window.Call("addEventListener", "blur", js.FuncOf(p.windowBlur))
	window.Call("addEventListener", "focus", js.FuncOf(p.windowFocus))
	p.UiComponents.paymentWindow = window.Call("open", url, "_blank")
}

// checkWindow checks if the new browser window/tab is still open. //Not sure if we need to do this?????????
func (p *ItemEditor) CheckWindow(window js.Value, callBack func()) {
	checkWindow := func() {
		for {
			if window.Get("closed").Bool() {
				break
			}
			time.Sleep(1 * time.Second)
		}
		log.Println(debugTag+"PaymentView.checkWindow()4", "window closed")
		callBack()
	}

	go checkWindow()
}

// onMakePaymentClick is triggerd by the user clicking the make payment button
// This triggers the start of the payment session
func (p *ItemEditor) MakePayment(ItemID int64) {
	var records []TableData
	var paymentURL string
	//var err error
	//p.ItemID = ItemID
	//err := p.StoreService.BookingReport.MakePayment(p.ItemID, p.openPaymentPage)
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"PaymentView.onMakePaymentClick()2", "err =", err)
	//	//return
	//}
	//p.Callback()

	success := func(err error, data *httpProcessor.ReturnData) {
		if data != nil {
			p.FieldNames = data.FieldNames // Might be able to use this to filter the fields displayed on the form
		}

		if records != nil {
			p.Records = records
		} else {
			p.Records = []TableData{}
			log.Println(debugTag + "FetchItems()1 records == nil")
		}

		p.populateItemList()
		p.updateStateDisplay(viewHelpers.ItemStateNone)
		p.openPaymentPage(paymentURL, nil)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		//log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Make Payment failed****", "err =", err, "len(Items) =", len(s.Items))
		log.Printf("%vgetServerVerify() fail: %v", debugTag, err)
		//editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
	}

	//p.client.NewRequest(http.MethodGet, ApiURL+"checkoutSession/create/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &records, nil, success, fail)
	p.client.NewRequest(http.MethodPost, ApiURL+"/create/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &records, nil, success, fail)

	//err := s.Client.SendGetRequest("checkoutSession/create/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &paymentURL, success, fail) //Send the REST request
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"Store.MakePayment()2 ", "err =", err) //Log the error in the browser
	//	return errors.New(debugTag + "MakePayment: what is the error description?????")
	//}
}

// windowClosed takes the next action in the payment process - advises the server that the payment page is complete
// The server then needs to check the payment status
func (p *ItemEditor) paymentWindowDestroy() {
	//var err error
	//Remove eventListners focus/blur
	//err := p.StoreService.BookingReport.ClosePayment(p.ItemID, nil)
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"PaymentView.paymentWindowDestroy()2", "err =", err)
	//	//return
	//}
	//p.Callback()

	//callbackSuccess := func(errIn error) {
	success := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v", debugTag+"Store.ClosePayment()1")
	}

	//callbackFail := func(errIn error) {
	fail := func(err error, data *httpProcessor.ReturnData) {
		//log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Close Payment failed****", "err =", err, "len(Items) =", len(s.Items))
		log.Printf("%v %v %v", debugTag+"PaymentView.paymentWindowDestroy()2", "err =", err)
	}

	p.client.NewRequest(http.MethodGet, ApiURL+"/closed/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), nil, nil, success, fail)

	//err := s.Client.SendGetRequest("checkoutSession/closed/"+strconv.FormatInt(BookingID, 10), nil, success, fail) //Send the REST request
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"Store.ClosePayment()2 ", "err =", err) //Log the error in the browser
	//	return errors.New(debugTag + "ClosePayment: what is the error description?????")
	//}
	//return nil

}

// checkPayment trigger payment check
// This is done by calling the server (api) to tell the server we might be finished.
// if payment complete we can close the window and do any other updates
func (p *ItemEditor) checkPayment() {
	//p.StoreService.BookingReport.CheckPayment(p.ParentItem.ID, callback)
	//p.StoreService.BookingReport.CheckPayment(p.ItemID, callback)

	var response string
	//success := func(err error) {
	success := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %+v", debugTag+"ListView.checkPayment()2", "err =", err)
		//switch err.Error() {
		switch response {
		case "open":
			//send open info to browser client
		case "complete":
			//send completed info to browser client
			p.UiComponents.paymentWindow.Call("close")
			p.paymentWindowDestroy()
		case "expired":
			//send expired info to browser client
			p.UiComponents.paymentWindow.Call("close")
			p.paymentWindowDestroy()
		}
	}

	//fail := func(errIn error) {
	fail := func(err error, data *httpProcessor.ReturnData) {
		//log.Printf("%v %v %v %v %v", debugTag+"Store.CheckPayment()4 ****Close Payment failed****", "err =", err, "len(Items) =", len(s.Items))
		log.Printf("%v %v %v", debugTag+"Store.CheckPayment()2 ", "err =", err) //Log the error in the browser
	}

	p.client.NewRequest(http.MethodGet, ApiURL+"/check/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &response, nil, success, fail)

	//err := s.Client.SendGetRequest("checkoutSession/check/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &response, success, fail) //Send the REST request
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"Store.CheckPayment()2 ", "err =", err) //Log the error in the browser
	//	return errors.New(debugTag + "ClosePayment: what is the error description?????")
	//}
	//return nil

}
*/

/*

package storeBookingReport

import (
	"errors"
	"log"
	"strconv"
)

// PaymentView displays the payment gateway page
type PaymentView struct {
	vecty.Core

	//Properties
	ItemID       int64          `vecty:"prop"`
	StoreService *store.Service `vecty:"prop"`
	Callback     func()         `vecty:"prop"`

	//State
	paymentWindow js.Value
}

func New(s *store.Service, callback func()) *PaymentView {
	return &PaymentView{
		StoreService: s,
		Callback:     callback,
	}
}

//MakePayment tells the server to contact the payment gateway with the information for setting up the payment
//and return the payment page url to the callback function.
func (s *Store) MakePayment(BookingID int64, callback func(string, error)) error { //int64 {
	var paymentURL string
	callbackSuccess := func(errIn error) {
		log.Printf("%v %v %v", debugTag+"Store.MakePayment()1", "paymentURL  =", paymentURL)
		if callback != nil {
			callback(paymentURL, errIn)
		} else {
			log.Println(debugTag + "MakePayment: callback is nil")
		}
	}

	callbackFail := func(errIn error) {
		log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Make Payment failed****", "errIn =", errIn, "len(Items) =", len(s.Items))
	}

	err := s.Client.SendGetRequest("checkoutSession/create/"+strconv.FormatInt(BookingID, 10), &paymentURL, callbackSuccess, callbackFail) //Send the REST request
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Store.MakePayment()2 ", "err =", err) //Log the error in the browser
		return errors.New(debugTag + "MakePayment: what is the error description?????")
	}
	return nil
}

//ClosePayment
func (s *Store) ClosePayment(BookingID int64, callback func(error)) error { //int64 {
	callbackSuccess := func(errIn error) {
		log.Printf("%v", debugTag+"Store.ClosePayment()1")
		if callback != nil {
			callback(errIn)
		} else {
			log.Println(debugTag + "ClosePayment: callback is nil")
		}
	}

	callbackFail := func(errIn error) {
		log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Close Payment failed****", "errIn =", errIn, "len(Items) =", len(s.Items))
	}

	err := s.Client.SendGetRequest("checkoutSession/closed/"+strconv.FormatInt(BookingID, 10), nil, callbackSuccess, callbackFail) //Send the REST request
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Store.ClosePayment()2 ", "err =", err) //Log the error in the browser
		return errors.New(debugTag + "ClosePayment: what is the error description?????")
	}
	return nil
}

//CheckPayment
func (s *Store) CheckPayment(BookingID int64, callback func(error)) error { //int64 {
	var response string
	callbackSuccess := func(errIn error) {
		log.Printf("%v", debugTag+"Store.CheckPayment()1")
		if callback != nil {
			callback(errors.New(response))
		} else {
			log.Println(debugTag + "CheckPayment: callback is nil")
		}
	}

	callbackFail := func(errIn error) {
		log.Printf("%v %v %v %v %v", debugTag+"Store.CheckPayment()4 ****Close Payment failed****", "errIn =", errIn, "len(Items) =", len(s.Items))
	}

	err := s.Client.SendGetRequest("checkoutSession/check/"+strconv.FormatInt(BookingID, 10), &response, callbackSuccess, callbackFail) //Send the REST request
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Store.CheckPayment()2 ", "err =", err) //Log the error in the browser
		return errors.New(debugTag + "ClosePayment: what is the error description?????")
	}
	return nil
}



// func (c *Client) SendGetRequest(url string, dataStru interface{}, callBacks ...func(error)) error {
*/
