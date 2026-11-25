package handlerBooking

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/bookings"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/bookings"
	"api-server/v2/models"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

const debugTag = "handlerBooking."

type Handler struct {
	appConf  *appCore.Config
	store    store.Store
	personal bool
}

func New(app *appCore.Config, personal ...bool) *Handler {
	h := &Handler{appConf: app, store: store.Store{DB: app.Db}}
	if len(personal) > 0 {
		h.personal = personal[0]
	}
	return h
}
func (h *Handler) RegisterRoutes(r *mux.Router, booking, trip string) {
	helpers.AddRouteGroup(r, booking, h, "bookings")
	helpers.CapabilityRoute(r, booking+"/checkout/reconcile/{id:[0-9]+}", "bookings", h.CheckoutReconcile).Methods("POST")
	h.RegisterRoutesStripe(r, booking)
	helpers.CapabilityRoute(r, trip+"/{id:[0-9]+}"+booking, "bookings", h.GetList).Methods("GET")
}
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) { h.list(w, r, 0, 0) }
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := helpers.RouteID(w, r)
	if id > 0 {
		h.list(w, r, 0, id)
	}
}
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	id := helpers.RouteID(w, r)
	if id > 0 {
		h.list(w, r, id, 0)
	}
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request, trip, id int) {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}
	page, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	if h.personal && trip == 0 && id == 0 {
		items, info, err := h.store.ListMy(r.Context(), session.UserID, page)
		if err != nil {
			helpers.WriteDataError(w, err)
			return
		}
		helpers.WriteResourcePage(w, items, info)
		return
	}
	items, info, err := h.store.List(r.Context(), session.UserID, strings.EqualFold(session.AccessScope, "any"), trip, id, page)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteResourcePage(w, items, info)
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.write(w, r, "create") }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.write(w, r, "update") }
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) { h.write(w, r, "delete") }
func (h *Handler) write(w http.ResponseWriter, r *http.Request, operation string) {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}
	any := strings.EqualFold(session.AccessScope, "any")
	var input models.Booking
	if operation != "delete" {
		var err error
		input, err = helpers.DecodeBookingWrite(w, r, any, operation == "create")
		if err != nil {
			helpers.WriteError(w, 400, "invalid_request", "Invalid request payload")
			return
		}
	}
	if operation != "create" {
		input.ID = helpers.RouteID(w, r)
		if input.ID <= 0 {
			return
		}
	}
	command := service.Service{Store: h.store, AllowTripChange: h.personal}
	if h.personal {
		command.Email = h.appConf.EmailService()
		command.ActorEmail = session.Email
	}
	result, err := command.Save(r.Context(), session.UserID, any, operation, input)
	if err != nil {
		if errors.Is(err, service.ErrVoucher) {
			helpers.WriteError(w, 422, "invalid_voucher", "Voucher code is invalid or expired")
		} else if errors.Is(err, service.ErrValidation) {
			helpers.WriteError(w, 422, "invalid_booking", "Invalid booking dates or trip reference")
		} else {
			helpers.WriteDataError(w, err)
		}
		return
	}
	if operation == "delete" {
		w.WriteHeader(200)
		return
	}
	if operation == "create" {
		helpers.WriteJSON(w, 201, result.ID)
		return
	}
	helpers.WriteJSON(w, 200, result)
}
