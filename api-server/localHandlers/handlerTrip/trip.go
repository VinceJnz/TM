package handlerTrip

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/trips"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/trips"
	"api-server/v2/models"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

const debugTag = "handlerTrip."

type Handler struct {
	appConf *appCore.Config
	store   store.Store
}

func New(app *appCore.Config) *Handler { return &Handler{appConf: app, store: store.Store{DB: app.Db}} }
func (h *Handler) RegisterRoutes(r *mux.Router, base string) {
	helpers.AddRouteGroup(r, base, h, "trips")
	helpers.CapabilityRoute(r, base+"Report", "tripsReport", h.GetParticipantStatus).Methods("GET")
	helpers.CapabilityRoute(r, base+"/{id:[0-9]+}/questionnaire", "trips", h.getQuestionnaire).Methods("GET")
	helpers.CapabilityRoute(r, base+"/{id:[0-9]+}/questionnaire", "trips", h.publishQuestionnaire).Methods("PUT")
}
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	page, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	items, info, err := h.store.List(r.Context(), page)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteResourcePage(w, items, info)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	item, err := h.store.Get(r.Context(), id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, []models.Trip{item})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.write(w, r, "create") }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.write(w, r, "update") }
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) { h.write(w, r, "delete") }
func (h *Handler) write(w http.ResponseWriter, r *http.Request, operation string) {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}
	var input models.Trip
	if operation != "delete" {
		var payload models.TripWrite
		if err := helpers.DecodeJSONBody(r, &payload); err != nil {
			helpers.WriteError(w, 400, "invalid_request", "Invalid request payload")
			return
		}
		input = payload.Trip()
	}
	if operation != "create" {
		input.ID = helpers.RouteID(w, r)
		if input.ID <= 0 {
			return
		}
	}
	result, err := (service.Service{Store: h.store}).Save(r.Context(), session.UserID, strings.EqualFold(session.AccessScope, "any"), operation, input)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			helpers.WriteError(w, 422, "invalid_trip", "Invalid trip dates, capacity or existing booking range")
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
