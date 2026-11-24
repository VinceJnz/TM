package handlerBookingPeople

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/participants"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/participants"
	"api-server/v2/models"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
)

type Handler struct {
	app   *appCore.Config
	store store.Store
}

func New(app *appCore.Config) *Handler { return &Handler{app, store.Store{DB: app.Db}} }
func (h *Handler) RegisterRoutes(r *mux.Router, path string) {
	helpers.AddRouteGroup(r, path, h, "bookingPeople")
	helpers.CapabilityRoute(r, "/bookings/{id:[0-9]+}/bookingPeople", "bookingPeople", h.GetList).Methods("GET")
	helpers.CapabilityRoute(r, "/bookings/{id:[0-9]+}/bookingPeople/questionnaire", "bookingPeople", h.getQuestionnaire).Methods("GET")
	helpers.CapabilityRoute(r, path+"/{id:[0-9]+}/details", "bookingPeople", h.getDetails).Methods("GET")
	helpers.CapabilityRoute(r, "/tripsReport/participants/{id:[0-9]+}/details", "tripsReport", h.getLeaderDetails).Methods("GET")
}
func (h *Handler) session(w http.ResponseWriter, r *http.Request) *models.Session {
	return helpers.RequiredSession(w, r, h.app.SessionIDKey)
}
func (h *Handler) read(w http.ResponseWriter, r *http.Request, parent, id int) {
	s := h.session(w, r)
	if s == nil {
		return
	}
	p, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	rows, info, err := h.store.List(r.Context(), s.UserID, s.AccessScope, parent, id, p)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteResourcePage(w, rows, info)
}
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) { h.read(w, r, 0, 0) }
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	id := helpers.RouteID(w, r)
	if id > 0 {
		h.read(w, r, id, 0)
	}
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := helpers.RouteID(w, r)
	if id > 0 {
		h.read(w, r, 0, id)
	}
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }
func (h *Handler) save(w http.ResponseWriter, r *http.Request, update bool) {
	session := h.session(w, r)
	if session == nil {
		return
	}
	id := 0
	if update {
		id = helpers.RouteID(w, r)
		if id == 0 {
			return
		}
	}
	var input models.ParticipantWrite
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid participant details")
		return
	}
	result, err := (service.Service{Store: h.store}).Mutate(r.Context(), session.UserID, session.AccessScope, id, input, false)
	if err != nil {
		h.failure(w, err)
		return
	}
	if !update {
		helpers.WriteJSON(w, 201, result)
		return
	}
	updated, err := h.store.Get(r.Context(), session.UserID, session.AccessScope, result)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, updated)
}
func (h *Handler) failure(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrLocked) {
		helpers.WriteError(w, 409, "booking_locked", "Booking is locked or inaccessible")
	} else if errors.Is(err, service.ErrValidation) {
		helpers.WriteError(w, 422, "invalid_participant", "Check the required participant details")
	} else if errors.Is(err, models.ErrQuestionnaire) {
		helpers.WriteError(w, 422, "invalid_questionnaire_answers", "Check the required questions and available answers")
	} else {
		helpers.WriteDataError(w, err)
	}
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	session := h.session(w, r)
	if session == nil {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	if _, err := (service.Service{Store: h.store}).Mutate(r.Context(), session.UserID, session.AccessScope, id, models.ParticipantWrite{}, true); err != nil {
		h.failure(w, err)
		return
	}
	helpers.WriteJSON(w, 200, nil)
}

const qryCreate = store.CreateQuery
