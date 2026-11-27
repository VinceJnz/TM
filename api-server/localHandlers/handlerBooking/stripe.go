package handlerBooking

import (
	"api-server/v2/app/services/checkout"
	"api-server/v2/app/services/notifications"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/modelMethods/paymentLedger"
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
	"github.com/stripe/stripe-go/v84/webhook"
)

//const debugTag = "handlerBooking."

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
	helpers.CapabilityRoute(r, baseURL+"/checkout/review/{id:[0-9]+}", "bookings", h.CheckoutReview).Methods("GET")
	helpers.CapabilityRoute(r, baseURL+"/checkout/confirm/{id:[0-9]+}", "bookings", h.CheckoutConfirm).Methods("POST")
	helpers.CapabilityRoute(r, baseURL+"/checkout/create/{id:[0-9]+}", "bookings", h.CheckoutCreate).Methods("POST")
	helpers.CapabilityRoute(r, baseURL+"/checkout/check/{id:[0-9]+}", "bookings", h.CheckoutCheck).Methods("GET")
	helpers.CapabilityRoute(r, baseURL+"/checkout/success/{id:[0-9]+}", "bookings", h.CheckoutSuccess).Methods("GET")
	helpers.CapabilityRoute(r, baseURL+"/checkout/cancel/{id:[0-9]+}", "bookings", h.CheckoutCancel).Methods("GET")
}

func (h *Handler) CheckoutReview(w http.ResponseWriter, r *http.Request) {
	user := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if user == nil {
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	review, err := paymentLedger.ReviewBooking(r.Context(), h.appConf.Db, id, user.UserID)
	if err != nil {
		writeBookingCheckoutError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, review)
}

func (h *Handler) CheckoutConfirm(w http.ResponseWriter, r *http.Request) {
	user := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if user == nil {
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	ctx := paymentLedger.WithAudit(r.Context(), "user:"+strconv.Itoa(user.UserID), "confirmed zero-total booking")
	if err := paymentLedger.ConfirmFreeBooking(ctx, h.appConf.Db, id, user.UserID); err != nil {
		writeBookingCheckoutError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{"confirmed"})
}

func writeBookingCheckoutError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, paymentLedger.ErrVoucherUnavailable):
		helpers.WriteError(w, http.StatusConflict, "invalid_voucher", "Voucher code is invalid or expired. Edit the booking to replace or remove it.")
	case errors.Is(err, paymentLedger.ErrQuoteChanged):
		helpers.WriteError(w, http.StatusConflict, "booking_total_changed", "The booking total changed. Review the booking again.")
	case errors.Is(err, paymentLedger.ErrCapacity):
		helpers.WriteError(w, http.StatusConflict, "capacity_unavailable", "The trip has insufficient space for this booking.")
	case errors.Is(err, paymentLedger.ErrParticipantDetails):
		helpers.WriteError(w, http.StatusConflict, "participant_details_required", "Complete the required details for each participant before continuing.")
	case errors.Is(err, paymentLedger.ErrBookingPricing):
		helpers.WriteError(w, http.StatusConflict, "booking_pricing_unavailable", "Pricing is unavailable for this booking. Ask the trip organiser to check the rates for the selected participants.")
	default:
		helpers.WriteDataError(w, err)
	}
}

// RegisterRoutesStripeWebhook registers the public Stripe webhook endpoint.
func (h *Handler) RegisterRoutesStripeWebhook(r *mux.Router, baseURL string) {
	helpers.CapabilityRoute(r, baseURL+"/checkout/webhook", "bookings", h.CheckoutWebhook).Methods("POST")
}

// CheckoutCreate This handler collects the data from the client and creates a payment intent.
// The client requests a checkout session by sending the booking ID
// This sets up a checkout session with stripe and sends the stripe checkout url to the client
func (h *Handler) CheckoutCreate(w http.ResponseWriter, r *http.Request) {
	if h.appConf.PaymentSvc == nil || h.appConf.PaymentSvc.Client == nil {
		http.Error(w, "payment service unavailable", 503)
		return
	}
	user := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if user == nil {
		return
	}
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	var expected []int64
	if r.Body != nil && r.Body != http.NoBody {
		var input struct {
			AmountMinor *int64 `json:"amount_minor"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			helpers.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid checkout request")
			return
		}
		if input.AmountMinor != nil {
			expected = append(expected, *input.AmountMinor)
		}
	}
	result, err := checkout.New(h.appConf.Db, h.appConf.PaymentSvc).Start(r.Context(), id, user.UserID, user.Email, h.appConf.PaymentSvc.Domain, expected...)
	if err != nil {
		if errors.Is(err, paymentLedger.ErrQuoteChanged) || errors.Is(err, paymentLedger.ErrVoucherUnavailable) ||
			errors.Is(err, paymentLedger.ErrCapacity) || errors.Is(err, paymentLedger.ErrParticipantDetails) {
			writeBookingCheckoutError(w, err)
			return
		}
		code := 503
		if errors.Is(err, paymentLedger.ErrCheckoutUnavailable) || errors.Is(err, paymentLedger.ErrAttemptExpired) || errors.Is(err, paymentLedger.ErrCapacity) {
			code = 409
		}
		http.Error(w, "checkout unavailable; reconcile the booking or retry", code)
		return
	}
	helpers.WriteJSON(w, 200, CheckoutCreateResponse{CheckoutURL: result.URL, SessionID: result.ID, Status: "created"})
}
func (h *Handler) CheckoutReconcile(w http.ResponseWriter, r *http.Request) {
	user := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if user == nil {
		return
	}
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	if h.appConf.PaymentSvc == nil || h.appConf.PaymentSvc.Client == nil {
		http.Error(w, "payment service unavailable", 503)
		return
	}
	outcome, err := checkout.New(h.appConf.Db, h.appConf.PaymentSvc).Reconcile(r.Context(), id, user.UserID)
	if err != nil {
		http.Error(w, "checkout reconciliation unavailable", 409)
		return
	}
	helpers.WriteJSON(w, 200, map[string]string{"status": outcome})
}
func (h *Handler) CheckoutCheck(w http.ResponseWriter, r *http.Request) {
	if h.appConf.PaymentSvc == nil || h.appConf.PaymentSvc.Client == nil {
		http.Error(w, "Payment service unavailable", http.StatusServiceUnavailable)
		return
	}
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	user := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if user == nil {
		http.Error(w, "booking not found", http.StatusNotFound)
		return
	}
	result, err := checkout.New(h.appConf.Db, h.appConf.PaymentSvc).Check(r.Context(), id, user.UserID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		http.Error(w, "booking not found", http.StatusNotFound)
	case errors.Is(err, checkout.ErrNoSession):
		http.Error(w, "No Stripe session found for booking", http.StatusNotFound)
	case err != nil:
		log.Printf("%sCheckoutCheck: %v", debugTag, err)
		http.Error(w, "Error retrieving checkout session", http.StatusInternalServerError)
	case result.State == "open", result.State == "complete", result.State == "expired":
		response := CheckoutStatusResponse{Status: result.State}
		if result.State == "complete" {
			response.AmountTotal = float64(result.Amount) / 100
			response.PaymentDate = &result.CheckedAt
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("%sCheckoutCheck response: %v", debugTag, err)
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
	if bookingID <= 0 {
		return
	}
	if h.appConf.TestMode {
		appSession := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
		if appSession != nil {
			if err := notifications.SendTestPaymentSuccess(h.appConf.EmailService(), appSession.Email); err != nil {
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
	<link rel="stylesheet" href="/tm-base-theme.css">
	<link rel="stylesheet" href="/tm-components.css">
	<link rel="stylesheet" href="/site-theme.css">
	<link rel="icon" href="/favicon.ico">
	<script src="/site-brand.js"></script>
	<script src="/site-branding.js" defer></script>
</head>
<body class="tm-checkout-result" data-page-title="Payment Successful">
<header class="navbar"><a href="/" class="tm-public-brand">Trip Manager</a></header>
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
	<main class="tm-member-card tm-card">
		<div class="tm-checkout-success" aria-hidden="true">&#10003;</div>
		<h1>Payment Completed Successfully</h1>
		<p>Your booking has been confirmed.</p>
		<p><a href="/#/mybookings">Return to My Bookings</a></p>
	</main>
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

	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
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

	if h.appConf.Settings.StripeTestMode && event.Livemode {
		http.Error(w, "Live Stripe events are disabled by test mode", http.StatusBadRequest)
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

	if event.Type != "checkout.session.completed" && event.Type != "checkout.session.async_payment_succeeded" {
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

	if h.appConf.Settings.StripeTestMode && checkoutSession.Livemode {
		http.Error(w, "Live checkout sessions are disabled by test mode", http.StatusBadRequest)
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

	var paidAt time.Time
	if event.Created > 0 {
		paidAt = time.Unix(event.Created, 0)
	}
	replay, err := checkout.New(h.appConf.Db, h.appConf.PaymentSvc).FinalizeWebhook(r.Context(), event.ID, checkout.Session{
		ID: checkoutSession.ID, AttemptID: checkoutSession.Metadata["tm_attempt"],
		State: string(checkoutSession.Status), Currency: string(checkoutSession.Currency),
		Mode: string(checkoutSession.Mode), Amount: checkoutSession.AmountTotal,
		Paid: checkoutSession.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid,
	}, paidAt)
	switch {
	case errors.Is(err, checkout.ErrUnpaidWebhook), errors.Is(err, checkout.ErrWebhookCurrency):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case errors.Is(err, checkout.ErrWebhookRecovery):
		http.Error(w, "checkout recovery failed", http.StatusInternalServerError)
		return
	case err != nil:
		logWebhookEvent("finalization_failed", map[string]interface{}{"event_id": event.ID, "checkout_session_id": checkoutSession.ID})
		http.Error(w, "payment finalization failed; retry required", http.StatusInternalServerError)
		return
	}
	logWebhookEvent("finalized", map[string]interface{}{"event_id": event.ID, "checkout_session_id": checkoutSession.ID, "replay_detected": replay})
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
	if bookingID <= 0 {
		return
	}
	// Return HTML page with cancellation message
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Payment Cancelled</title>
	<link rel="stylesheet" href="/tm-base-theme.css">
	<link rel="stylesheet" href="/tm-components.css">
	<link rel="stylesheet" href="/site-theme.css">
	<link rel="icon" href="/favicon.ico">
	<script src="/site-brand.js"></script>
	<script src="/site-branding.js" defer></script>
</head>
<body class="tm-checkout-result" data-page-title="Payment Cancelled">
<header class="navbar"><a href="/" class="tm-public-brand">Trip Manager</a></header>
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
	<main class="tm-member-card tm-card">
		<div class="tm-checkout-cancel" aria-hidden="true">&#10005;</div>
		<h1>Payment Cancelled</h1>
		<p>Your payment was not completed.</p>
		<p><a href="/#/mybookings">Return to My Bookings</a></p>
	</main>
</body>
</html>`, bookingID)
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("%v failed to write cancel HTML response: %v", debugTag+"CheckoutCancel()", err)
	}
}
