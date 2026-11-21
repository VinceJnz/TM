package handlerUser

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/accounts"
	profiles "api-server/v2/app/services/profiles"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/accounts"
	profileStore "api-server/v2/modelMethods/profiles"
	"api-server/v2/models"
	"errors"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	appConf *appCore.Config
	store   store.Store
}

func New(app *appCore.Config) *Handler { return &Handler{app, store.Store{DB: app.Db}} }
func (h *Handler) RegisterRoutes(r *mux.Router, path string) {
	helpers.AddRouteGroup(r, path, h, "users")
	helpers.CapabilityRoute(r, path+"/set-username", "users", h.SetUsername).Methods("POST")
	helpers.CapabilityRoute(r, path+"/lookup", "users", h.Lookup).Methods("GET")
}
func (h *Handler) actor(w http.ResponseWriter, r *http.Request) int {
	s := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if s == nil {
		return 0
	}
	if !strings.EqualFold(s.AccessScope, "any") {
		helpers.WriteError(w, 403, "forbidden", "Account administration requires any scope")
		return 0
	}
	return s.UserID
}
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	if h.actor(w, r) == 0 {
		return
	}
	p, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	rows, info, err := h.store.List(r.Context(), p)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteResourcePage(w, rows, info)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if h.actor(w, r) == 0 {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	v, err := h.store.Get(r.Context(), id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, []models.AccountDTO{v})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }
func (h *Handler) save(w http.ResponseWriter, r *http.Request, update bool) {
	actor := h.actor(w, r)
	if actor == 0 {
		return
	}
	id := 0
	if update {
		id = helpers.RouteID(w, r)
		if id == 0 {
			return
		}
	}
	var input models.AccountWrite
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid security payload")
		return
	}
	result, err := (service.Service{Store: h.store}).Save(r.Context(), actor, id, input, r.Header.Get("X-Change-Reason"))
	if err != nil {
		h.failure(w, err)
		return
	}
	if !update {
		if err := helpers.NotifyAdminsUserReviewRequired("account-create", h.appConf.Db, h.appConf.EmailService(), h.appConf.Settings.AdminNotifyGroup, result.ID, result.Username, result.Email, result.Name); err != nil {
			log.Print("account review notification unavailable")
		}
		helpers.WriteJSON(w, 201, result.ID)
		return
	}
	helpers.WriteJSON(w, 200, result)
}
func (h *Handler) failure(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrReferenced) {
		helpers.WriteError(w, 409, "account_in_use", "Account has dependent records; disable it instead")
	} else if errors.Is(err, service.ErrReason) {
		helpers.WriteError(w, 422, "change_reason_required", "Provide a reason for account recovery, status changes or deletion")
	} else if errors.Is(err, service.ErrValidation) {
		helpers.WriteError(w, 422, "invalid_account", "Invalid account values")
	} else {
		helpers.WriteDataError(w, err)
	}
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor := h.actor(w, r)
	if actor == 0 {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	if err := (service.Service{Store: h.store}).Delete(r.Context(), actor, id, r.Header.Get("X-Change-Reason")); err != nil {
		h.failure(w, err)
		return
	}
	helpers.WriteJSON(w, 200, nil)
}

func (h *Handler) SetUsername(w http.ResponseWriter, r *http.Request) {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}
	var input struct {
		Username string `json:"username"`
	}
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid username request")
		return
	}
	username, err := (profiles.Service{Store: profileStore.Store{DB: h.appConf.Db}}).SetUsername(r.Context(), session.UserID, input.Username)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			helpers.WriteError(w, 400, "invalid_username", "Username must contain 3 to 20 bytes")
		} else {
			helpers.WriteDataError(w, err)
		}
		return
	}
	helpers.WriteJSON(w, 200, map[string]string{"status": "ok", "username": username})
}

func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	if h.actor(w, r) == 0 {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	id := 0
	if raw := r.URL.Query().Get("id"); raw != "" {
		var err error
		id, err = strconv.Atoi(raw)
		if err != nil || id < 1 {
			helpers.WriteError(w, 400, "invalid_id", "Invalid user ID")
			return
		}
	}
	if len(query) > 100 {
		helpers.WriteError(w, 400, "invalid_search", "Search must be at most 100 bytes")
		return
	}
	items, err := h.store.Lookup(r.Context(), query, id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, items)
}
