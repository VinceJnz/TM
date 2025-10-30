package stripe

import (
	"encoding/json"
	"log"
	"os"

	"github.com/stripe/stripe-go/v72/client"
)

const debugTag = "stripe."

//https://dashboard.stripe.com/test/dashboard
//https://stripe.com/docs/checkout/quickstart
//https://stripe.com/docs/api/checkout/sessions/create#create_checkout_session-line_items-price_data
//https://stripe.com/docs/api/payment_intents

/*
Payment succeeds  4242 4242 4242 4242
Payment requires authentication  4000 0025 0000 3155
Payment is declined  4000 0000 0000 9995
*/

type Charge struct {
	Amount       int64  `json:"amount"`
	ReceiptEmail string `json:"receiptMail"`
	ProductName  string `json:"productName"`
}

type Gateway struct {
	//appConf    *appCore.Config
	PaymentSvc *client.API
	domain     string
}

// func New(appConf *appCore.Config, keyFile, domain string) *Gateway {
func New(keyFile, domain string) *Gateway {
	return &Gateway{
		PaymentSvc: client.New(KeyFromFile(keyFile), nil),
		domain:     domain,
		//appConf:    appConf,
	}
}

/*
const (
	qryGet = `SELECT id, owner_id, trip_id, notes, from_date, to_date, booking_status_id, ebs.status, ab.booking_date, ab.payment_date, ab.booking_price, created, modified
					FROM at_bookings WHERE id = $1`

	qryCreate = `INSERT INTO at_bookings (owner_id, trip_id, notes, from_date, to_date, booking_status_id)
        			VALUES ($1, $2, $3, $4, $5, $6)
					RETURNING id`
)

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGet, id)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Booking
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	//log.Printf(debugTag+"Create()3 record = %+v, query = %+v", record, qryCreate)

	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, session.UserID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID)
}

func (h *Handler) RecordValidation(record models.Booking) error {
	if err := helpers.ValidateDatesFromLtTo(record.FromDate, record.ToDate); err != nil {
		return err
	}
	if err := h.ParentRecordValidation(record); err != nil {
		return err
	}
	return nil
}

const (
	sqlBookingParentRecordValidation = `SELECT * FROM at_trips WHERE id = $1`
)

func (h *Handler) ParentRecordValidation(record models.Booking) error {
	parentID := record.TripID

	validationRecord := models.Trip{}
	err := h.appConf.Db.Get(&validationRecord, sqlBookingParentRecordValidation, parentID)
	if err == sql.ErrNoRows {
		return fmt.Errorf(debugTag+"ParentRecordValidation()1 - Record not found: error message = %s, parentID = %v", err.Error(), parentID)
	} else if err != nil {
		return fmt.Errorf(debugTag+"ParentRecordValidation()2 - Internal Server Error:  error message = %s, parentID = %v", err.Error(), parentID)
	}

	if record.FromDate.Before(validationRecord.FromDate) {
		return fmt.Errorf("dateError: Booking From-date is before Trip From-date")
	}

	if record.ToDate.After(validationRecord.ToDate) {
		return fmt.Errorf("dateError: Booking To-date is after Trip To-date")
	}

	return nil
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

	h.Get(recordID, h.Table)
	bookingItem := h.Table.(*mdlBooking.Item)
	//If the number of paid up people on the trip exceeds the trip capacity then do not allow the payment.
	if (bookingItem.TripPersonCount.Int64 + bookingItem.BookingPersonCount.Int64) > bookingItem.TripMaxPeople.Int64 {
		//Need to return some sort of error message to the client
		msg := "payment disallowed: The booking will exceed the trip capacity"
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.CheckoutCreate()1", "msg =", msg, "bookingItem", bookingItem)
		http.Error(w, msg, http.StatusConflict)
		return
	}

	if bookingItem.Cost.Float64 == 0 {
		//Need to return some sort of error message to the client
		msg := "payment disallowed: The booking cost is zero"
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.CheckoutCreate()1", "msg =", msg, "bookingItem", bookingItem)
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
						Description: stripe.String("Trip description: " + bookingItem.TripDescription.String),
						//Images:      []*string{},
						//Metadata:    map[string]string{},
						Name: stripe.String("Trip name = " + bookingItem.TripName.String),
						//TaxCode:     new(string),
					},
					//Recurring:         &stripe.CheckoutSessionLineItemPriceDataRecurringParams{},
					//TaxBehavior:       new(string),
					UnitAmount: stripe.Int64(int64(bookingItem.Cost.ValueOrZero() * 100)), //Amount in cents
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
		SuccessURL: stripe.String(h.domain + "/checkoutSession/success/" + strconv.FormatInt(recordID, 10)),
		CancelURL:  stripe.String(h.domain + "/checkoutSession/cancel/" + strconv.FormatInt(recordID, 10)),
	}
	CheckoutSession, err = h.PaymentSvc.CheckoutSessions.New(params)
	if err != nil {
		log.Printf("session.New: %v", err)
	}
	log.Printf("%v %v %v %v %+v %v %v", debugTag+"Handler.CheckoutCreate()2", "CheckoutSession.ID =", CheckoutSession.ID, "CheckoutSession =", CheckoutSession, "recordID =", recordID)

	//Update the Booking record with the stripe checkout session id
	bookingItem.PaymentToken.SetValid(CheckoutSession.ID)
	bookingItem.TotalFee.SetValid(bookingItem.Cost.Float64)
	h.Put(recordID, bookingItem)

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
	var recordID int64

	log.Printf("%v", debugTag+"Handler.CheckoutCheck()1")

	vars := mux.Vars(r)
	recordID, err = strconv.ParseInt(vars["recordID"], 10, 64)
	if err != nil {
		err = errors.New("invalid or missing record id")
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.Get(recordID, h.Table)
	bookingItem := h.Table.(*mdlBooking.Item)

	CheckoutSession, err := h.PaymentSvc.CheckoutSessions.Get(bookingItem.PaymentToken.String, nil) //get the active stripe checkout session from the stripe server
	if err != nil {
		log.Printf("session.Update: %v", err)
	}
	log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.CheckoutCheck()2", "bookingItem.ID =", bookingItem.ID, "CheckoutSession.ID =", CheckoutSession.ID, "CheckoutSession =", CheckoutSession) //, "h.wsPool =", h.wsPool)

	//Check the payment status
	switch CheckoutSession.Status {
	case "open":
		//send open info to browser client
		json.NewEncoder(w).Encode("open")
	case "complete":
		//Update the booking record to show that the payment is complete
		bookingItem.PaidStatusID.SetValid(mdlBooking.Full_amount)
		bookingItem.AmountPaid.SetValid(float64(CheckoutSession.AmountTotal) / 100) //???????
		bookingItem.DatePaid.SetValid(time.Now())
		h.Put(recordID, bookingItem)

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

// CheckoutClose this handler is called by the payment window if the payment session has been cancelled.
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
*/

// KeyFromFile Get the stripe authentication token from a local file.
func KeyFromFile(file string) string {
	f, err := os.Open(file)
	if err != nil {
		//log.Fatalf("%v %v %v", debugTag+"main()1", "err =", err)
		log.Printf("%v %v %v", debugTag+"KeyFromFile(()1", "err =", err)
		return ""
	}
	defer f.Close()
	var key map[string]string
	err = json.NewDecoder(f).Decode(&key)
	if err != nil {
		//log.Fatalf("%v %v %v", debugTag+"main()1", "err =", err)
		log.Printf("%v %v %+v %v %v", debugTag+"KeyFromFile(()2", "key =", key, "err =", err)
	}
	if keyValue, ok := key["key"]; ok {
		return keyValue
	}
	log.Printf("%v %v %+v %v %v", debugTag+"KeyFromFile(()3", "key =", key, "err =", err)
	return ""
}
