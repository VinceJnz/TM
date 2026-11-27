package handlerUserPayments

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/services/finance"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/paymentLedger"
	"api-server/v2/modelMethods/payments"
	"api-server/v2/models"

	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const debugTag = "handlerUserPayments."

type Handler struct {
	appConf *appCore.Config
	store   payments.Store
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf, store: payments.Store{DB: appConf.Db}}
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h, "payments")
	helpers.CapabilityRoute(r, baseURL+"/booking/{id:[0-9]+}", "payments", h.GetList).Methods("GET")
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	h.list(w, r, 0)
}

func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	h.list(w, r, id)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	item, err := h.store.Get(r.Context(), id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	// Preserve the established single-record array response.
	helpers.WriteJSON(w, 200, []models.UserPayments{item})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, booking int) {
	limit, offset := 100, 0
	for key, target := range map[string]*int{"limit": &limit, "offset": &offset} {
		if raw := r.URL.Query().Get(key); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				helpers.WriteError(w, 400, "invalid_pagination", "limit must be 1..500 and offset 0..1000000")
				return
			}
			*target = n
		}
	}
	if limit < 1 || limit > 500 || offset < 0 || offset > 1000000 {
		helpers.WriteError(w, 400, "invalid_pagination", "limit must be 1..500 and offset 0..1000000")
		return
	}
	var items []models.UserPayments
	var more bool
	var err error
	if r.URL.Query().Get("pagination") == "cursor" {
		after := int64(0)
		if raw := r.URL.Query().Get("after"); raw != "" {
			after, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || after < 0 {
				helpers.WriteError(w, 400, "invalid_pagination", "limit must be 1..500 and offset 0..1000000")
				return
			}
		}
		items, more, err = h.store.ListAfter(r.Context(), booking, limit, after)
		w.Header().Set("X-Pagination-Mode", "cursor")
		if more && len(items) > 0 {
			w.Header().Set("X-Next-Cursor", strconv.Itoa(items[len(items)-1].ID))
		}
	} else {
		items, more, err = h.store.List(r.Context(), booking, limit, offset)
	}
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	w.Header().Set("X-Page-Limit", strconv.Itoa(limit))
	if more {
		w.Header().Set("X-Next-Offset", strconv.Itoa(offset+limit))
	}
	helpers.WriteJSON(w, 200, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.write(w, r, "create") }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.write(w, r, "update") }
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) { h.write(w, r, "delete") }
func (h *Handler) write(w http.ResponseWriter, r *http.Request, operation string) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	id := 0
	if operation != "create" {
		id = helpers.RouteID(w, r)
		if id <= 0 {
			return
		}
	}
	var input paymentLedger.PaymentWrite
	if operation != "delete" {
		if err := helpers.DecodeJSONBody(r, &input); err != nil {
			helpers.WriteError(w, 400, "invalid_request", "Invalid request payload")
			return
		}
	}
	session := r.Context().Value(h.appConf.SessionIDKey).(*models.Session)
	id, err := (finance.Service{DB: h.appConf.Db}).Payment(r.Context(), session.UserID, r.Header.Get("X-Change-Reason"), operation, id, input)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	if operation == "delete" {
		w.WriteHeader(200)
		return
	}
	if operation == "create" {
		helpers.WriteJSON(w, 201, id)
		return
	}
	response := models.UserPayments{ID: id, BookingID: input.BookingID, PaymentDate: input.PaymentDate, Amount: input.Amount, PaymentMethod: input.PaymentMethod}
	helpers.WriteJSON(w, 200, response)
}
