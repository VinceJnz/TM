package handlerBooking

import (
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
)

//const debugTag = "handlerBooking."

const (
	qryGetBookingForPayment = `SELECT 
		atb.id, atb.owner_id, atb.trip_id, atb.from_date, atb.to_date, 
		atb.booking_status_id, atb.stripe_session_id, atb.amount_paid,
		att.trip_name, att.description, att.max_participants,
		COUNT(atbp.id) as participants,
		SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
		(SELECT COUNT(*) FROM at_booking_people atbp2 
			JOIN at_bookings atb2 ON atbp2.booking_id = atb2.id 
			WHERE atb2.trip_id = att.id AND atb2.booking_status_id = 2) as trip_person_count
	FROM at_bookings atb
	JOIN at_trips att ON att.id = atb.trip_id
	LEFT JOIN at_booking_people atbp ON atbp.booking_id = atb.id
	LEFT JOIN st_users stu ON stu.id = atbp.person_id
	LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id = att.trip_cost_group_id
		AND attc.member_status_id = stu.member_status_id
		AND attc.user_age_group_id = stu.user_age_group_id
	WHERE atb.id = $1
	GROUP BY atb.id, att.id, att.trip_name, att.description, att.max_participants`

	qryUpdateStripeSession = `UPDATE at_bookings 
		SET stripe_session_id = $2, booking_price = $3
		WHERE id = $1`

	qryUpdatePaymentComplete = `UPDATE at_bookings 
		SET booking_status_id = $2, amount_paid = $3, payment_date = $4
		WHERE id = $1`
)

// Response structures for consistent API responses
type CheckoutCreateResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
	Status      string `json:"status"`
}

type CheckoutStatusResponse struct {
	Status      string     `json:"status"`
	AmountTotal float64    `json:"amount_total,omitempty"`
	PaymentDate *time.Time `json:"payment_date,omitempty"`
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutesStripe(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/checkout/create/{id:[0-9]+}", h.CheckoutCreate).Methods("POST")
	r.HandleFunc(baseURL+"/checkout/check/{id:[0-9]+}", h.CheckoutCheck).Methods("GET")
	r.HandleFunc(baseURL+"/checkout/success/{id:[0-9]+}", h.CheckoutSuccess).Methods("GET")
	r.HandleFunc(baseURL+"/checkout/cancel/{id:[0-9]+}", h.CheckoutCancel).Methods("GET")
}

// CheckoutCreate This handler collects the data from the client and creates a payment intent.
// The client requests a checkout session by sending the booking ID
// This sets up a checkout session with stripe and sends the stripe checkout url to the client
func (h *Handler) CheckoutCreate(w http.ResponseWriter, r *http.Request) { //, s *mdlSession.Item) {
	var err error
	//var recordID int64
	var CheckoutSession *stripe.CheckoutSession
	appSession := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	log.Printf("%v, appSession: %+v", debugTag+"Handler.CheckoutCreate()1", appSession)

	// Need to create a structure and queries here to get the booking and trip details
	// which neeeds to be passed to stripe to create the checkout session
	// Trip details include trip name, trip cost, trip Max Participants etc.
	// Booking details include booking id, booking person, place in booking queue etc.
	// compare with Participant Status query in tripParticipantStatus.go ????

	// Get booking ID from URL parameter
	bookingID := dbStandardTemplate.GetID(w, r)
	if bookingID == 0 {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	// Fetch booking details
	bookingItem := &models.BookingPaymentInfo{}
	err = Get(w, r, debugTag, h.appConf.Db, bookingItem, qryGetBookingForPayment, bookingID)
	if err != nil {
		log.Printf("%v %v %v, booking item = %+v", debugTag+"Handler.CheckoutCreate()2", "Get() error =", err, bookingItem)
		return
	}

	// Validate booking can be paid
	if err := h.validateBookingForPayment(bookingItem, int64(appSession.UserID)); err != nil {
		log.Printf("%v validation error: %v", debugTag+"Handler.CheckoutCreate()3", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// Create Stripe checkout session
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			//&stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencyNZD)),
					//Product:  stripe.String("BookingID:" + strconv.FormatInt(bookingItem.ID, 10)), //This could be a booking reference that can be used to find all the payment records associated with a booking record
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Description: stripe.String("Trip description: " + bookingItem.Description.String),
						//Images:      []*string{},
						//Metadata:    map[string]string{},
						Name: stripe.String("Trip name = " + bookingItem.TripName.String),
						//TaxCode:     new(string),
					},
					//Recurring:         &stripe.CheckoutSessionLineItemPriceDataRecurringParams{},
					//TaxBehavior:       new(string),
					UnitAmount: stripe.Int64(int64(bookingItem.BookingCost.ValueOrZero() * 100)), //Amount in cents
					//UnitAmountDecimal: new(float64),
				},
				Quantity: stripe.Int64(1),
			},
		},
		//CustomerEmail: stripe.String(s.User.Email.String),
		CustomerEmail: stripe.String("vince.jennings@gmail.com"), // Debug/POC only
		Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:    stripe.String(h.appConf.PaymentSvc.Domain + "/bookings/checkout/success/" + strconv.Itoa(bookingID)),
		CancelURL:     stripe.String(h.appConf.PaymentSvc.Domain + "/bookings/checkout/cancel/" + strconv.Itoa(bookingID)),

		// Add this for invoice emails (works in test mode):
		InvoiceCreation: &stripe.CheckoutSessionInvoiceCreationParams{
			Enabled: stripe.Bool(true),
		},
	}

	// NEW WAY: Use session.New with the client
	CheckoutSession, err = session.New(params)
	if err != nil {
		log.Printf("%v session.New error: %v", debugTag+"Handler.CheckoutCreate()4", err)
		http.Error(w, "Error creating checkout session", http.StatusInternalServerError)
		return
	}
	log.Printf("%v CheckoutSession.ID = %v, CheckoutSession = %+v, bookingID = %v", debugTag+"Handler.CheckoutCreate()5", CheckoutSession.ID, CheckoutSession, bookingID)

	//Update the Booking record with the stripe checkout session id
	bookingItem.StripeSessionID.SetValid(CheckoutSession.ID)
	//bookingItem.TotalFee.SetValid(bookingItem.Cost.Float64) //??????? is this recording the amount paid or is it just a flag???
	err = Update(w, r, debugTag, h.appConf.Db, bookingItem, qryUpdateStripeSession, bookingItem.ID, bookingItem.StripeSessionID, bookingItem.BookingCost)
	if err != nil {
		log.Printf("%v %v %v, booking item = %+v", debugTag+"Handler.CheckoutCreate()6", "Update() error =", err, bookingItem)
		return
	}
	//*******************************************************
	//r.Header.Set("Access-Control-Allow-Origin", "stripe.com, 111.stripe.com")
	//w.Header().Set("Access-Control-Allow-Origin", "stripe.com, 222.stripe.com")
	//log.Printf("%v %v %+v %v %+v", debugTag+"Handler.CheckoutCreate()6", "w.Header() =", w.Header(), "r =", r)
	log.Printf("%v %v %v", debugTag+"Handler.CheckoutCreate()7", "CheckoutSession.URL", CheckoutSession.URL)

	// Return structured JSON response
	response := CheckoutCreateResponse{
		CheckoutURL: CheckoutSession.URL,
		SessionID:   CheckoutSession.ID,
		Status:      "created",
	}

	//send the stripe checkout session url to the browser client
	//http.Redirect(w, r, CheckoutSession.URL, http.StatusSeeOther)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckoutCheck verifies the current status of a Stripe checkout session
// CheckoutCheck is called when the browser client receives focus or some other appropriate trigger.
// This checks the status of the stripe checkout session with the stripe server
// The browser client can then take action, e.g. close the payment window and update the payment report.
// or do nothing and continue to wait for the payment to complete
func (h *Handler) CheckoutCheck(w http.ResponseWriter, r *http.Request) { //, s *mdlSession.Item) {
	var err error

	log.Printf("%v", debugTag+"Handler.CheckoutCheck()1")

	bookingID := dbStandardTemplate.GetID(w, r)

	// Fetch booking details
	bookingItem := &models.BookingPaymentInfo{}
	err = Get(w, r, debugTag, h.appConf.Db, bookingItem, qryGetBookingForPayment, bookingID)
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Handler.CheckoutCheck()2", "Get() error =", err)
		return
	}
	if !bookingItem.StripeSessionID.Valid || bookingItem.StripeSessionID.String == "" {
		log.Printf("%v No Stripe session found for booking %d", debugTag+"Handler.CheckoutCheck()3", bookingID)
		http.Error(w, "No Stripe session found for booking", http.StatusNotFound)
		return
	}

	// Get checkout session from Stripe
	// NEW WAY: Use session.New with the client
	CheckoutSession, err := session.Get(bookingItem.StripeSessionID.String, nil)
	if err != nil {
		log.Printf("%v session.Get error: %v, bookingItem = %+v", debugTag+"Handler.CheckoutCheck()3", err, bookingItem)
		http.Error(w, "Error retrieving checkout session", http.StatusInternalServerError)
		return
	}
	log.Printf("%v CheckoutSession = %+v, bookingID = %d", debugTag+"Handler.CheckoutCheck()4", CheckoutSession, bookingItem.ID)

	// Build response based on session status
	response := CheckoutStatusResponse{
		Status: string(CheckoutSession.Status),
	}

	// Handle payment completion
	//Check the payment status
	switch CheckoutSession.Status {
	case stripe.CheckoutSessionStatusOpen: //"open":
		//send open info to browser client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	case stripe.CheckoutSessionStatusComplete: //"complete":
		//Update the booking record to show that the payment is complete
		bookingItem.BookingStatusID.SetValid(int64(models.Full_amountPaid)) //Payment status = Full amount paid (value is 2) and sould only be set if the full payment has been made
		bookingItem.AmountPaid.SetValid(CheckoutSession.AmountTotal)        //Store the amount paid
		bookingItem.DatePaid.SetValid(time.Now())
		err = Update(w, r, debugTag, h.appConf.Db, bookingItem, qryUpdatePaymentComplete, bookingItem.ID, bookingItem.BookingStatusID, bookingItem.AmountPaid, bookingItem.DatePaid)
		if err != nil {
			log.Printf("%v %v %v, booking item = %+v", debugTag+"Handler.CheckoutCheck()5", "Update() error =", err, bookingItem)
			http.Error(w, "Error updating booking payment status", http.StatusInternalServerError)
			return
		}
		response.AmountTotal = float64(CheckoutSession.AmountTotal) / 100
		response.PaymentDate = &bookingItem.DatePaid.Time
		//send completed info to browser client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	case stripe.CheckoutSessionStatusExpired: // "expired":
		//send expired info to browser client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}

	//poc ??????????? websocket poc stuff ??????????????????????
	//h.wsPool.FindKey(bookingItem.PaymentToken.String).Write([]byte("complete"))
	//h.wsPool.FindKey(strconv.FormatInt(bookingItem.ID, 10)).Write([]byte("complete"))
}

// CheckoutSuccess handles successful payment redirect
// CheckoutSuccess this handler is called by the payment window if the payment session has been successful.
// All it does is provide a message to the payment window to say it can be closed.
// When the browser client detects that it has focus or if the payment window has been closed the client can take further action
// Note: The browser is not logged in so we can't guarantee the information supplied by the browser
func (h *Handler) CheckoutSuccess(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v", debugTag+"CheckoutSuccess()1")

	// User validation passed ????
	//h.validateUser( &models.BookingPaymentInfo{}, 0)

	// Get booking ID from URL parameter
	//recordID := dbStandardTemplate.GetID(w, r) // THis is not used ????
	//if recordID == 0 {
	//	http.Error(w, "Invalid booking ID", http.StatusBadRequest)
	//	return
	//}

	if h.appConf.TestMode {
		appSession := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
		h.appConf.EmailSvc.SendMail("vince.jennings@gmail.com", "Test Mode - Payment Successful", "Test mode enabled - payment successful for uers email:"+appSession.Email)
	}

	//log.Printf("%v Payment successful for booking %d", debugTag+"CheckoutSuccess()3", recordID)

	//Send a completed page to the payment window/tab
	// Return HTML page with success message
	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Payment Successful</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
			background-color: #f5f5f5;
		}
		.container {
			text-align: center;
			padding: 40px;
			background: white;
			border-radius: 8px;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
		}
		.success-icon {
			color: #4caf50;
			font-size: 48px;
			margin-bottom: 20px;
		}
		h1 {
			color: #333;
			margin-bottom: 10px;
		}
		p {
			color: #666;
			margin-bottom: 20px;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="success-icon">✓</div>
		<h1>Payment Completed Successfully</h1>
		<p>Your booking has been confirmed.</p>
		<p>You can now close this window.</p>
	</div>
</body>
</html>`
	w.Write([]byte(html))
}

// CheckoutCancel handles cancelled payment redirect
// CheckoutCancel this handler is called by the payment window if the payment session has been cancelled.
// All it does is provide a message to the payment window to say it can be closed.
// When the browser client detects that it has focus or if the payment window has been closed the client can take further action
// Note: The browser is not logged in so we can't guarantee the information supplied by the browser
func (h *Handler) CheckoutCancel(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v", debugTag+"CheckoutCancel()1")

	// Get booking ID from URL parameter
	//recordID := dbStandardTemplate.GetID(w, r) // THis is not used ????
	//if recordID == 0 {
	//	http.Error(w, "Invalid booking ID", http.StatusBadRequest)
	//	return
	//}

	//log.Printf("%v Payment cancelled for booking %d", debugTag+"CheckoutCancel()3", recordID)

	// Return HTML page with cancellation message
	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Payment Cancelled</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
			background-color: #f5f5f5;
		}
		.container {
			text-align: center;
			padding: 40px;
			background: white;
			border-radius: 8px;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
		}
		.cancel-icon {
			color: #ff9800;
			font-size: 48px;
			margin-bottom: 20px;
		}
		h1 {
			color: #333;
			margin-bottom: 10px;
		}
		p {
			color: #666;
			margin-bottom: 20px;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="cancel-icon">✕</div>
		<h1>Payment Cancelled</h1>
		<p>Your payment was not completed.</p>
		<p>You can now close this window.</p>
	</div>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// validateBookingForPayment checks if booking can proceed to payment
func (h *Handler) validateBookingForPayment(bookingItem *models.BookingPaymentInfo, appSessionUserID int64) error {
	// Check if booking will exceed trip capacity
	if bookingItem.BookingPosition.Int64+bookingItem.BookingParticipants.Int64 > bookingItem.MaxParticipants.Int64 {
		return errors.New("payment disallowed: the booking will exceed the trip capacity")
	}

	// Check if booking cost is valid
	if bookingItem.BookingCost.Float64 == 0 {
		return errors.New("payment disallowed: the booking cost is zero")
	}

	// User validation passed ????
	if bookingItem.OwnerID != appSessionUserID {
		return errors.New("unauthorized: user is not the booking owner")
	}

	return nil
}

func (h *Handler) validateUser(bookingItem *models.BookingPaymentInfo, appSessionUserID int64) error {

	// User validation passed ????
	if bookingItem.OwnerID != appSessionUserID {
		return errors.New("unauthorized: user is not the booking owner")
	}
	return nil
}

/*

	//vars := mux.Vars(r)
	//recordID, err = strconv.ParseInt(vars["id"], 10, 64)
	//if err != nil {
	//	err = errors.New("invalid or missing record id")
	//	http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
	//	return
	//}


    //If the number of paid up people on the trip exceeds the trip capacity then do not allow the payment.
	if bookingItem.BookingPosition.Int64+bookingItem.BookingParticipants.Int64 > bookingItem.MaxPeople.Int64 {
		//Need to return some sort of error message to the client
		msg := "payment disallowed: The booking will exceed the trip capacity"
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.CheckoutCreate()1", "msg =", msg, "bookingItem", bookingItem)
		http.Error(w, msg, http.StatusConflict)
		return
	}

	if bookingItem.BookingCost.Float64 == 0 {
		//Need to return some sort of error message to the client
		msg := "payment disallowed: The booking cost is zero"
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.CheckoutCreate()2", "msg =", msg, "bookingItem", bookingItem)
		http.Error(w, msg, http.StatusConflict)
		return
	}
*/
