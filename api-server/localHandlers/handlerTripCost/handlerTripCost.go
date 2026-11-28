package handlerTripCost

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/pricing"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/pricing"
	"api-server/v2/models"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

type Handler struct {
	app   *appCore.Config
	store store.Store
}

func New(app *appCore.Config) *Handler { return &Handler{app, store.Store{DB: app.Db}} }
func (h *Handler) RegisterRoutes(r *mux.Router, path string) {
	helpers.AddRouteGroup(r, path, h, "tripCosts")
}
func (h *Handler) allowed(w http.ResponseWriter, r *http.Request) bool {
	s := helpers.RequiredSession(w, r, h.app.SessionIDKey)
	if s == nil {
		return false
	}
	if !strings.EqualFold(s.AccessScope, "any") {
		helpers.WriteError(w, 403, "forbidden", "Pricing requires any scope")
		return false
	}
	return true
}
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	p, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	groupID, valid := helpers.OptionalQueryID(w, r, "trip_cost_group_id")
	if !valid {
		return
	}
	items, info, err := h.store.Costs(r.Context(), p, groupID)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteResourcePage(w, items, info)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	v, err := h.store.Cost(r.Context(), id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, []models.TripCost{v})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }
func (h *Handler) save(w http.ResponseWriter, r *http.Request, update bool) {
	if !h.allowed(w, r) {
		return
	}
	id := 0
	if update {
		id = helpers.RouteID(w, r)
		if id == 0 {
			return
		}
	}
	var input models.TripCostWrite
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid pricing payload")
		return
	}
	v, err := (service.Service{Store: h.store}).SaveCost(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			helpers.WriteError(w, 400, "invalid_request", "Invalid pricing values")
		} else {
			helpers.WriteDataError(w, err)
		}
		return
	}
	if !update {
		helpers.WriteJSON(w, 201, v.ID)
		return
	}
	helpers.WriteJSON(w, 200, v)
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	if err := h.store.DeleteCost(r.Context(), id); err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, nil)
}
