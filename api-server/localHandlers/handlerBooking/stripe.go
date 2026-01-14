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
		att.trip_name, att.description, att.max_people,
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
	GROUP BY atb.id, att.id, att.trip_name, att.description, att.max_people`

	qryUpdateStripeSession = `UPDATE at_bookings 
		SET stripe_session_id = $2, booking_price = $3
		WHERE id = $1`

	qryUpdatePaymentComplete = `UPDATE at_bookings 
		SET booking_status_id = $2, amount_paid = $3, payment_date = $4
		WHERE id = $1`
)

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
	var recordID int64
	var CheckoutSession *stripe.CheckoutSession
	log.Printf("%v", debugTag+"Handler.CheckoutCreate()1")

	vars := mux.Vars(r)
	recordID, err = strconv.ParseInt(vars["recordID"], 10, 64)
	if err != nil {
		err = errors.New("invalid or missing record id")
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Need to create a structure and queries here to get the booking and trip details
	// which neeeds to be passed to stripe to create the checkout session
	// Trip details include trip name, trip cost, trip Max Participants etc.
	// Booking details include booking id, booking person, place in booking queue etc.
	// compare with Participant Status query in tripParticipantStatus.go ????
	bookingItem := &models.BookingPaymentInfo{}
	bookingID := dbStandardTemplate.GetID(w, r)
	Get(w, r, debugTag, h.appConf.Db, bookingItem, qryGetBookingForPayment, bookingID)
	log.Printf("%v %v %+v", debugTag+"Handler.CheckoutCreate()1", "bookingItem =", bookingItem)

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
		CustomerEmail: stripe.String("vince.jennings@gmail.com"),
		Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
		//SuccessURL:    stripe.String(h.domain + "/checkoutSession/success/"),
		//CancelURL:     stripe.String(h.domain + "/checkoutSession/cancel/"),
		SuccessURL: stripe.String(h.appConf.PaymentSvc.Domain + "/checkoutSession/success/" + strconv.FormatInt(recordID, 10)),
		CancelURL:  stripe.String(h.appConf.PaymentSvc.Domain + "/checkoutSession/cancel/" + strconv.FormatInt(recordID, 10)),
	}

	// NEW WAY: Use session.New with the client
	CheckoutSession, err = session.New(params)
	if err != nil {
		log.Printf("session.New: %v", err)
		http.Error(w, "Error creating checkout session", http.StatusInternalServerError)
		return
	}
	log.Printf("%v %v %v %v %+v %v %v", debugTag+"Handler.CheckoutCreate()2", "CheckoutSession.ID =", CheckoutSession.ID, "CheckoutSession =", CheckoutSession, "recordID =", recordID)

	//Update the Booking record with the stripe checkout session id
	bookingItem.StripeSessionID.SetValid(CheckoutSession.ID)
	//bookingItem.TotalFee.SetValid(bookingItem.Cost.Float64) //??????? is this recording the amount paid or is it just a flag???
	Update(w, r, debugTag, h.appConf.Db, bookingItem, qryUpdateStripeSession, bookingItem.ID, bookingItem.StripeSessionID, bookingItem.BookingCost)

	//*******************************************************
	r.Header.Set("Access-Control-Allow-Origin", "stripe.com, 111.stripe.com")
	w.Header().Set("Access-Control-Allow-Origin", "stripe.com, 222.stripe.com")
	log.Printf("%v %v %+v %v %+v", debugTag+"Handler.CheckoutCreate()3", "w.Header() =", w.Header(), "r =", r)
	log.Printf("%v %v %v", debugTag+"Handler.CheckoutCreate()4", "CheckoutSession.URL", CheckoutSession.URL)

	//send the stripe checkout session url to the browser client
	//http.Redirect(w, r, CheckoutSession.URL, http.StatusSeeOther)
	json.NewEncoder(w).Encode(CheckoutSession.URL)
}

// CheckoutCheck is called when the browser client receives focus or some other appropriate trigger.
// This checks the status of the stripe checkout session with the stripe server
// The browser client can then take action, e.g. close the payment window and update the payment report.
// or do nothing and continue to wait for the payment to complete
func (h *Handler) CheckoutCheck(w http.ResponseWriter, r *http.Request) { //, s *mdlSession.Item) {
	var err error

	log.Printf("%v", debugTag+"Handler.CheckoutCheck()1")

	bookingItem := &models.BookingPaymentInfo{}
	bookingID := dbStandardTemplate.GetID(w, r)
	Get(w, r, debugTag, h.appConf.Db, bookingItem, qryGetBookingForPayment, bookingID)
	log.Printf("%v %v %+v", debugTag+"Handler.CheckoutCheck()1", "bookingItem =", bookingItem)

	// NEW WAY: Use session.New with the client
	CheckoutSession, err := session.Get(bookingItem.StripeSessionID.String, nil)
	if err != nil {
		log.Printf("session.New: %v", err)
		http.Error(w, "Error creating checkout session", http.StatusInternalServerError)
		return
	}
	log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.CheckoutCheck()2", "bookingItem.ID =", bookingItem.ID, "CheckoutSession.ID =", CheckoutSession.ID, "CheckoutSession =", CheckoutSession) //, "h.wsPool =", h.wsPool)

	//Check the payment status
	switch CheckoutSession.Status {
	case "open":
		//send open info to browser client
		json.NewEncoder(w).Encode("open")
	case "complete":
		//Update the booking record to show that the payment is complete
		bookingItem.PaymentStatusID.SetValid(int64(models.Full_amountPaid))
		bookingItem.AmountPaid.SetValid(float64(CheckoutSession.AmountTotal) / 100) //???????
		bookingItem.PaymentStatusID.SetValid(int64(models.Full_amountPaid))
		bookingItem.DatePaid.SetValid(time.Now())
		Update(w, r, debugTag, h.appConf.Db, bookingItem, qryUpdatePaymentComplete, bookingItem.ID, bookingItem.PaymentStatusID, bookingItem.AmountPaid, bookingItem.DatePaid)

		//send completed info to browser client
		json.NewEncoder(w).Encode("complete")
	case "expired":
		//send expired info to browser client
		json.NewEncoder(w).Encode("expired")
	}

	//poc ??????????? websocket poc stuff ??????????????????????
	//h.wsPool.FindKey(bookingItem.PaymentToken.String).Write([]byte("complete"))
	//h.wsPool.FindKey(strconv.FormatInt(bookingItem.ID, 10)).Write([]byte("complete"))
}

// CheckoutSuccess this handler is called by the payment window if the payment session has been successful.
// All it does is provide a message to the payment window to say it can be closed.
// When the browser client detects that it has focus or if the payment window has been closed the client can take further action
// Note: The browser is not logged in so we can't guarantee the information supplied by the browser
func (h *Handler) CheckoutSuccess(w http.ResponseWriter, r *http.Request) {
	var err error
	var recordID int64
	//var wsToken string

	log.Printf("%v", debugTag+"CheckoutClose()1")

	vars := mux.Vars(r)
	recordID, err = strconv.ParseInt(vars["recordID"], 10, 64)
	if err != nil {
		err = errors.New("invalid or missing record id")
		//http.Error(ws, "Error: "+err.Error(), http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v", debugTag+"Handler.Websocket()1", "err =", err, "recordID =", recordID)
		return
	}

	//Send a completed page to the payment window/tab
	html := `<!DOCTYPE html>
		<html>
		<head></head>
		<body>
		<div>Payment completed successfully</Div>
		<div>You can now close this payment tab</Div>
		</body>
		</html>`
	w.Write([]byte(html))
}

// CheckoutCancel this handler is called by the payment window if the payment session has been cancelled.
// All it does is provide a message to the payment window to say it can be closed.
// When the browser client detects that it has focus or if the payment window has been closed the client can take further action
// Note: The browser is not logged in so we can't guarantee the information supplied by the browser
func (h *Handler) CheckoutCancel(w http.ResponseWriter, r *http.Request) {
	var err error
	var recordID int64
	//var wsToken string

	log.Printf("%v", debugTag+"CheckoutClose()1")

	vars := mux.Vars(r)
	recordID, err = strconv.ParseInt(vars["recordID"], 10, 64)
	if err != nil {
		err = errors.New("invalid or missing record id")
		//http.Error(ws, "Error: "+err.Error(), http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v", debugTag+"Handler.Websocket()1", "err =", err, "recordID =", recordID)
		return
	}

	//Send a completed page to the payment window/tab
	html := `<!DOCTYPE html>
		<html>
		<head></head>
		<body>
		<div>Payment cancelled</Div>
		<div>You can now close this payment tab</Div>
		</body>
		</html>`
	w.Write([]byte(html))
}
