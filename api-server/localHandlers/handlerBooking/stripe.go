package handlerBooking

import (
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/webhook"
)

//const debugTag = "handlerBooking."

const (
	qryGetBookingForPayment = `SELECT 
		atb.id, atb.owner_id, atb.trip_id, atb.from_date, atb.to_date, atb.booking_price,
		atb.booking_status_id, atb.stripe_session_id, atb.amount_paid,
		att.trip_name, att.description, att.max_participants,
		COUNT(atbp.id) as participants,
		SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
		(SELECT COUNT(*) FROM at_booking_people atbp2 
			JOIN at_bookings atb2 ON atbp2.booking_id = atb2.id 
			WHERE atb2.trip_id = att.id AND atb2.booking_status_id = 3) as trip_person_count
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

	qryUpdatePaymentByStripeSession = `UPDATE at_bookings
		SET booking_status_id = $2, amount_paid = $3, payment_date = $4
		WHERE stripe_session_id = $1 AND booking_status_id <> $2`

	qryUpdatePaymentByBookingID = `UPDATE at_bookings
		SET booking_status_id = $2, amount_paid = $3, payment_date = $4
		WHERE id = $1 AND booking_status_id <> $2`

	qryUpsertPaymentByStripeSession = `INSERT INTO at_payments (booking_id, payment_date, amount, payment_method, stripe_session_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (stripe_session_id)
		DO UPDATE
		SET booking_id = EXCLUDED.booking_id,
			payment_date = EXCLUDED.payment_date,
			amount = EXCLUDED.amount,
			payment_method = EXCLUDED.payment_method,
			modified = CURRENT_TIMESTAMP`

	qryGetBookingByStripeSession = `SELECT id, owner_id, booking_status_id
		FROM at_bookings
		WHERE stripe_session_id = $1
		LIMIT 1`
)

func (h *Handler) upsertPaymentRecord(bookingID int64, amountCents int64, paidAt time.Time, stripeSessionID string) error {
	if bookingID <= 0 || stripeSessionID == "" {
		return errors.New("invalid payment upsert payload")
	}
	amount := float64(amountCents) / 100.0
	_, err := h.appConf.Db.Exec(qryUpsertPaymentByStripeSession, bookingID, paidAt, amount, "stripe_checkout", stripeSessionID)
	return err
}

func logWebhookEvent(stage string, fields map[string]interface{}) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["component"] = "stripe_webhook"
	fields["stage"] = stage
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	encoded, err := json.Marshal(fields)
	if err != nil {
		log.Printf("%vCheckoutWebhook() stage=%s marshal_err=%v fields=%+v", debugTag, stage, err, fields)
		return
	}
	log.Printf("%vCheckoutWebhook() %s", debugTag, string(encoded))
}

func parseStripeSignatureTimestamp(signatureHeader string) int64 {
	for _, part := range strings.Split(signatureHeader, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "t=") {
			continue
		}
		timestamp, err := strconv.ParseInt(strings.TrimPrefix(part, "t="), 10, 64)
		if err == nil {
			return timestamp
		}
	}
	return 0
}

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

// RegisterRoutesStripeWebhook registers the public Stripe webhook endpoint.
func (h *Handler) RegisterRoutesStripeWebhook(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/checkout/webhook", h.CheckoutWebhook).Methods("POST")
}

// CheckoutCreate This handler collects the data from the client and creates a payment intent.
// The client requests a checkout session by sending the booking ID
// This sets up a checkout session with stripe and sends the stripe checkout url to the client
func (h *Handler) CheckoutCreate(w http.ResponseWriter, r *http.Request) {
	var err error
	var CheckoutSession *stripe.CheckoutSession
	if h.appConf.PaymentSvc == nil || h.appConf.PaymentSvc.Client == nil {
		log.Printf("%v Stripe payment service unavailable", debugTag+"Handler.CheckoutCreate()")
		http.Error(w, "Payment service unavailable", http.StatusServiceUnavailable)
		return
	}
	appSession := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if appSession == nil || appSession.UserID == 0 {
		log.Printf("%v missing authenticated session", debugTag+"Handler.CheckoutCreate()")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("%v appSession=%+v", debugTag+"Handler.CheckoutCreate()", appSession)

	// Need to create a structure and queries here to get the booking and trip details
	// which neeeds to be passed to stripe to create the checkout session
	// Trip details include trip name, trip cost, trip Max Participants etc.
	// Booking details include booking id, booking person, place in booking queue etc.
	// Compare with participant-status behavior in tripParticipantStatus flow if this query changes.

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
		log.Printf("%v get booking failed err=%v booking=%+v", debugTag+"Handler.CheckoutCreate()", err, bookingItem)
		return
	}

	// Validate booking can be paid
	if err := h.validateBookingForPayment(bookingItem, int64(appSession.UserID)); err != nil {
		log.Printf("%v validation error: %v", debugTag+"Handler.CheckoutCreate()", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// Create Stripe checkout session
	chargeAmount := bookingItem.BookingCost.ValueOrZero()
	if bookingItem.BookingPrice.ValueOrZero() > 0 {
		chargeAmount = bookingItem.BookingPrice.ValueOrZero()
	}

	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencyNZD)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Description: stripe.String("Trip description: " + bookingItem.Description.String),
						Name:        stripe.String("Trip name = " + bookingItem.TripName.String),
					},
					UnitAmount: stripe.Int64(int64(chargeAmount * 100)), //Amount in cents
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(h.appConf.PaymentSvc.Domain + "/bookings/checkout/success/" + strconv.Itoa(bookingID)),
		CancelURL:  stripe.String(h.appConf.PaymentSvc.Domain + "/bookings/checkout/cancel/" + strconv.Itoa(bookingID)),

		// Add this for invoice emails (works in test mode):
		InvoiceCreation: &stripe.CheckoutSessionInvoiceCreationParams{
			Enabled: stripe.Bool(true),
		},
	}
	if appSession.Email != "" {
		params.CustomerEmail = stripe.String(appSession.Email)
	}

	// NEW WAY: Use session.New with the client
	CheckoutSession, err = session.New(params)
	if err != nil {
		log.Printf("%v session.New error: %v", debugTag+"Handler.CheckoutCreate()", err)
		http.Error(w, "Error creating checkout session", http.StatusInternalServerError)
		return
	}
	log.Printf("%v created session id=%v bookingID=%v", debugTag+"Handler.CheckoutCreate()", CheckoutSession.ID, bookingID)

	//Update the Booking record with the stripe checkout session id
	bookingItem.StripeSessionID.SetValid(CheckoutSession.ID)
	err = Update(w, r, debugTag, h.appConf.Db, bookingItem, qryUpdateStripeSession, bookingItem.ID, bookingItem.StripeSessionID, bookingItem.BookingCost)
	if err != nil {
		log.Printf("%v update booking stripe session failed err=%v booking=%+v", debugTag+"Handler.CheckoutCreate()", err, bookingItem)
		return
	}
	log.Printf("%v checkout_url=%v", debugTag+"Handler.CheckoutCreate()", CheckoutSession.URL)

	// Return structured JSON response
	response := CheckoutCreateResponse{
		CheckoutURL: CheckoutSession.URL,
		SessionID:   CheckoutSession.ID,
		Status:      "created",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("%v failed to write checkout create response: %v", debugTag+"Handler.CheckoutCreate()", err)
	}
}

// CheckoutCheck verifies the current status of a Stripe checkout session
// CheckoutCheck is called when the browser client receives focus or some other appropriate trigger.
// This checks the status of the stripe checkout session with the stripe server
// The browser client can then take action, e.g. close the payment window and update the payment report.
// or do nothing and continue to wait for the payment to complete
func (h *Handler) CheckoutCheck(w http.ResponseWriter, r *http.Request) {
	var err error
	if h.appConf.PaymentSvc == nil || h.appConf.PaymentSvc.Client == nil {
		log.Printf("%v Stripe payment service unavailable", debugTag+"Handler.CheckoutCheck()")
		http.Error(w, "Payment service unavailable", http.StatusServiceUnavailable)
		return
	}

	log.Printf("%v checking checkout status", debugTag+"Handler.CheckoutCheck()")

	bookingID := dbStandardTemplate.GetID(w, r)

	// Fetch booking details
	bookingItem := &models.BookingPaymentInfo{}
	err = Get(w, r, debugTag, h.appConf.Db, bookingItem, qryGetBookingForPayment, bookingID)
	if err != nil {
		log.Printf("%v get booking failed err=%v", debugTag+"Handler.CheckoutCheck()", err)
		return
	}
	if !bookingItem.StripeSessionID.Valid || bookingItem.StripeSessionID.String == "" {
		log.Printf("%v no stripe session found for booking=%d", debugTag+"Handler.CheckoutCheck()", bookingID)
		http.Error(w, "No Stripe session found for booking", http.StatusNotFound)
		return
	}

	// Get checkout session from Stripe
	CheckoutSession, err := session.Get(bookingItem.StripeSessionID.String, nil)
	if err != nil {
		log.Printf("%v session.Get error: %v booking=%+v", debugTag+"Handler.CheckoutCheck()", err, bookingItem)
		http.Error(w, "Error retrieving checkout session", http.StatusInternalServerError)
		return
	}
	log.Printf("%v stripe status=%s bookingID=%d", debugTag+"Handler.CheckoutCheck()", CheckoutSession.Status, bookingItem.ID)

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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("%v failed to write checkout status response (open): %v", debugTag+"Handler.CheckoutCheck()", err)
		}
	case stripe.CheckoutSessionStatusComplete: //"complete":
		if h.appConf.TestMode && h.appConf.Settings.StripeWebhookSecret == "" {
			// Dev-only fallback for local testing when webhook forwarding is not configured.
			// Production still relies on webhook-only finalization.
			updateRes, updateErr := h.appConf.Db.Exec(qryUpdatePaymentByBookingID, bookingItem.ID, int(models.Full_amountPaid), CheckoutSession.AmountTotal, time.Now())
			if updateErr != nil {
				logWebhookEvent("dev_fallback_db_update_failed", map[string]interface{}{
					"booking_id":          bookingItem.ID,
					"checkout_session_id": bookingItem.StripeSessionID.String,
					"error":               updateErr.Error(),
				})
			} else {
				if upsertErr := h.upsertPaymentRecord(int64(bookingItem.ID), CheckoutSession.AmountTotal, time.Now(), bookingItem.StripeSessionID.String); upsertErr != nil {
					logWebhookEvent("dev_fallback_payment_upsert_failed", map[string]interface{}{
						"booking_id":          bookingItem.ID,
						"checkout_session_id": bookingItem.StripeSessionID.String,
						"error":               upsertErr.Error(),
					})
				}
				rows, _ := updateRes.RowsAffected()
				logWebhookEvent("dev_fallback_applied", map[string]interface{}{
					"booking_id":          bookingItem.ID,
					"checkout_session_id": bookingItem.StripeSessionID.String,
					"rows_affected":       rows,
				})
			}
		}

		// Webhook is authoritative for DB updates. This endpoint only reports Stripe status.
		response.AmountTotal = float64(CheckoutSession.AmountTotal) / 100
		now := time.Now()
		response.PaymentDate = &now
		//send completed info to browser client
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("%v failed to write checkout status response (complete): %v", debugTag+"Handler.CheckoutCheck()", err)
		}
	case stripe.CheckoutSessionStatusExpired: // "expired":
		//send expired info to browser client
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("%v failed to write checkout status response (expired): %v", debugTag+"Handler.CheckoutCheck()", err)
		}
	}
}

// CheckoutSuccess handles successful payment redirect
// CheckoutSuccess this handler is called by the payment window if the payment session has been successful.
// All it does is provide a message to the payment window to say it can be closed.
// When the browser client detects that it has focus or if the payment window has been closed the client can take further action
// Note: The browser is not logged in so we can't guarantee the information supplied by the browser
func (h *Handler) CheckoutSuccess(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v handling checkout success", debugTag+"CheckoutSuccess()")
	bookingID := dbStandardTemplate.GetID(w, r)

	if h.appConf.TestMode && h.appConf.Settings.StripeWebhookSecret == "" && bookingID > 0 {
		bookingItem := &models.BookingPaymentInfo{}
		if err := Get(w, r, debugTag, h.appConf.Db, bookingItem, qryGetBookingForPayment, bookingID); err != nil {
			logWebhookEvent("dev_fallback_success_lookup_failed", map[string]interface{}{
				"booking_id": bookingID,
				"error":      err.Error(),
			})
		} else if bookingItem.StripeSessionID.Valid && bookingItem.StripeSessionID.String != "" {
			checkoutSession, err := session.Get(bookingItem.StripeSessionID.String, nil)
			if err != nil {
				logWebhookEvent("dev_fallback_success_session_get_failed", map[string]interface{}{
					"booking_id":          bookingID,
					"checkout_session_id": bookingItem.StripeSessionID.String,
					"error":               err.Error(),
				})
			} else if checkoutSession != nil && checkoutSession.Status == stripe.CheckoutSessionStatusComplete {
				updateRes, updateErr := h.appConf.Db.Exec(qryUpdatePaymentByBookingID, bookingID, int(models.Full_amountPaid), checkoutSession.AmountTotal, time.Now())
				if updateErr != nil {
					logWebhookEvent("dev_fallback_success_db_update_failed", map[string]interface{}{
						"booking_id":          bookingID,
						"checkout_session_id": bookingItem.StripeSessionID.String,
						"error":               updateErr.Error(),
					})
				} else {
					if upsertErr := h.upsertPaymentRecord(int64(bookingID), checkoutSession.AmountTotal, time.Now(), bookingItem.StripeSessionID.String); upsertErr != nil {
						logWebhookEvent("dev_fallback_success_payment_upsert_failed", map[string]interface{}{
							"booking_id":          bookingID,
							"checkout_session_id": bookingItem.StripeSessionID.String,
							"error":               upsertErr.Error(),
						})
					}
					rows, _ := updateRes.RowsAffected()
					logWebhookEvent("dev_fallback_success_applied", map[string]interface{}{
						"booking_id":          bookingID,
						"checkout_session_id": bookingItem.StripeSessionID.String,
						"rows_affected":       rows,
					})
				}
			}
		}
	}

	if h.appConf.TestMode {
		appSession := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
		if h.appConf.EmailSvc != nil && appSession != nil && appSession.Email != "" {
			if _, err := h.appConf.EmailSvc.SendMail(appSession.Email, "Test Mode - Payment Successful", "Test mode enabled - payment successful for user email:"+appSession.Email); err != nil {
				log.Printf("%v failed to send test-mode payment success email: %v", debugTag+"CheckoutSuccess()", err)
			}
		}
	}

	//Send a completed page to the payment window/tab
	// Return HTML page with success message
	html := fmt.Sprintf(`<!DOCTYPE html>
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
	<script>
		(function () {
			var payload = { type: "tm-payment-status", status: "complete", bookingId: %d };
			try {
				if (window.opener && !window.opener.closed) {
					window.opener.postMessage(payload, window.location.origin);
				}
			} catch (e) {
				try {
					window.opener.postMessage(payload, "*");
				} catch (_e) {}
			}
			setTimeout(function () { window.close(); }, 300);
		})();
	</script>
	<div class="container">
		<div class="success-icon">&#10003;</div>
		<h1>Payment Completed Successfully</h1>
		<p>Your booking has been confirmed.</p>
		<p>You can now close this window.</p>
	</div>
</body>
</html>`, bookingID)
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("%v failed to write success HTML response: %v", debugTag+"CheckoutSuccess()", err)
	}
}

// CheckoutWebhook handles Stripe webhook events and is the authoritative payment finalization path.
func (h *Handler) CheckoutWebhook(w http.ResponseWriter, r *http.Request) {
	signatureHeader := r.Header.Get("Stripe-Signature")
	signatureTimestamp := parseStripeSignatureTimestamp(signatureHeader)

	if h.appConf.Settings.StripeWebhookSecret == "" {
		logWebhookEvent("config_error", map[string]interface{}{
			"reason":              "webhook_secret_missing",
			"signature_timestamp": signatureTimestamp,
		})
		http.Error(w, "webhook not configured", http.StatusServiceUnavailable)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		logWebhookEvent("read_error", map[string]interface{}{
			"error":               err.Error(),
			"signature_timestamp": signatureTimestamp,
		})
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(payload, signatureHeader, h.appConf.Settings.StripeWebhookSecret)
	if err != nil {
		logWebhookEvent("signature_verification_failed", map[string]interface{}{
			"error":               err.Error(),
			"signature_timestamp": signatureTimestamp,
			"payload_size":        len(payload),
		})
		http.Error(w, "signature verification failed", http.StatusBadRequest)
		return
	}

	eventAgeSec := int64(0)
	if signatureTimestamp > 0 {
		eventAgeSec = time.Now().Unix() - signatureTimestamp
	}

	logWebhookEvent("received", map[string]interface{}{
		"event_id":            event.ID,
		"event_type":          event.Type,
		"event_created_unix":  event.Created,
		"livemode":            event.Livemode,
		"signature_timestamp": signatureTimestamp,
		"event_age_seconds":   eventAgeSec,
	})

	if event.Type != "checkout.session.completed" {
		logWebhookEvent("ignored_event", map[string]interface{}{
			"event_id":   event.ID,
			"event_type": event.Type,
		})
		w.WriteHeader(http.StatusOK)
		return
	}

	var checkoutSession stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &checkoutSession); err != nil {
		logWebhookEvent("parse_error", map[string]interface{}{
			"event_id":   event.ID,
			"event_type": event.Type,
			"error":      err.Error(),
		})
		http.Error(w, "invalid event payload", http.StatusBadRequest)
		return
	}

	if checkoutSession.ID == "" {
		logWebhookEvent("invalid_session", map[string]interface{}{
			"event_id":   event.ID,
			"event_type": event.Type,
			"reason":     "missing_checkout_session_id",
		})
		http.Error(w, "missing checkout session id", http.StatusBadRequest)
		return
	}

	res, err := h.appConf.Db.Exec(qryUpdatePaymentByStripeSession, checkoutSession.ID, int(models.Full_amountPaid), checkoutSession.AmountTotal, time.Now())
	if err != nil {
		logWebhookEvent("db_update_failed", map[string]interface{}{
			"event_id":            event.ID,
			"checkout_session_id": checkoutSession.ID,
			"error":               err.Error(),
		})
		http.Error(w, "db update failed", http.StatusInternalServerError)
		return
	}

	rows, _ := res.RowsAffected()

	replay := false
	replayReason := ""
	bookingID := int64(0)
	ownerID := int64(0)
	bookingStatus := int64(0)
	if rows == 0 {
		replay = true
		var rowBookingID sql.NullInt64
		var rowOwnerID sql.NullInt64
		var rowBookingStatus sql.NullInt64
		rowErr := h.appConf.Db.QueryRow(qryGetBookingByStripeSession, checkoutSession.ID).Scan(&rowBookingID, &rowOwnerID, &rowBookingStatus)
		switch {
		case rowErr == sql.ErrNoRows:
			replayReason = "no_booking_for_session"
		case rowErr != nil:
			replayReason = "lookup_error"
		default:
			if rowBookingID.Valid {
				bookingID = rowBookingID.Int64
			}
			if rowOwnerID.Valid {
				ownerID = rowOwnerID.Int64
			}
			if rowBookingStatus.Valid {
				bookingStatus = rowBookingStatus.Int64
			}
			if bookingStatus == int64(models.Full_amountPaid) {
				replayReason = "already_paid"
			} else {
				replayReason = "no_state_change"
			}
		}
	}

	if bookingID == 0 || ownerID == 0 {
		var rowBookingID sql.NullInt64
		var rowOwnerID sql.NullInt64
		var rowBookingStatus sql.NullInt64
		rowErr := h.appConf.Db.QueryRow(qryGetBookingByStripeSession, checkoutSession.ID).Scan(&rowBookingID, &rowOwnerID, &rowBookingStatus)
		if rowErr == nil {
			if rowBookingID.Valid {
				bookingID = rowBookingID.Int64
			}
			if rowOwnerID.Valid {
				ownerID = rowOwnerID.Int64
			}
			if rowBookingStatus.Valid {
				bookingStatus = rowBookingStatus.Int64
			}
		}
	}

	if bookingID > 0 {
		if upsertErr := h.upsertPaymentRecord(bookingID, checkoutSession.AmountTotal, time.Now(), checkoutSession.ID); upsertErr != nil {
			logWebhookEvent("payment_upsert_failed", map[string]interface{}{
				"event_id":            event.ID,
				"checkout_session_id": checkoutSession.ID,
				"booking_id":          bookingID,
				"error":               upsertErr.Error(),
			})
		}
	}

	logWebhookEvent("finalized", map[string]interface{}{
		"event_id":            event.ID,
		"event_type":          event.Type,
		"checkout_session_id": checkoutSession.ID,
		"booking_id":          bookingID,
		"owner_id":            ownerID,
		"booking_status_id":   bookingStatus,
		"rows_affected":       rows,
		"replay_detected":     replay,
		"replay_reason":       replayReason,
	})
	w.WriteHeader(http.StatusOK)
}

// CheckoutCancel handles cancelled payment redirect
// CheckoutCancel this handler is called by the payment window if the payment session has been cancelled.
// All it does is provide a message to the payment window to say it can be closed.
// When the browser client detects that it has focus or if the payment window has been closed the client can take further action
// Note: The browser is not logged in so we can't guarantee the information supplied by the browser
func (h *Handler) CheckoutCancel(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v handling checkout cancel", debugTag+"CheckoutCancel()")
	bookingID := dbStandardTemplate.GetID(w, r)

	// Return HTML page with cancellation message
	html := fmt.Sprintf(`<!DOCTYPE html>
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
	<script>
		(function () {
			var payload = { type: "tm-payment-status", status: "cancelled", bookingId: %d };
			try {
				if (window.opener && !window.opener.closed) {
					window.opener.postMessage(payload, window.location.origin);
				}
			} catch (e) {
				try {
					window.opener.postMessage(payload, "*");
				} catch (_e) {}
			}
			setTimeout(function () { window.close(); }, 300);
		})();
	</script>
	<div class="container">
		<div class="cancel-icon">&#10005;</div>
		<h1>Payment Cancelled</h1>
		<p>Your payment was not completed.</p>
		<p>You can now close this window.</p>
	</div>
</body>
</html>`, bookingID)
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("%v failed to write cancel HTML response: %v", debugTag+"CheckoutCancel()", err)
	}
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

	if bookingItem.OwnerID != appSessionUserID {
		return errors.New("unauthorized: user is not the booking owner")
	}

	return nil
}
