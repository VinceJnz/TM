package handlerMyProfile

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/profiles"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/profiles"
	"api-server/v2/models"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"time"
)

type ProfileResponse = models.ProfileDTO
type ProfileUpdateRequest = models.ProfileWrite
type Handler struct {
	appConf *appCore.Config
	store   store.Store
}

func New(app *appCore.Config) *Handler { return &Handler{app, store.Store{DB: app.Db}} }
func (h *Handler) RegisterRoutes(r *mux.Router, path string) {
	helpers.CapabilityRoute(r, path, "myProfile", h.Get).Methods("GET")
	helpers.CapabilityRoute(r, path, "myProfile", h.Update).Methods("PUT")
	helpers.CapabilityRoute(r, path+"/email-change", "myProfile", h.RequestEmail).Methods("POST")
	helpers.CapabilityRoute(r, path+"/email-change/confirm", "myProfile", h.ConfirmEmail).Methods("POST")
}
func (h *Handler) actor(w http.ResponseWriter, r *http.Request) int {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return 0
	}
	if !strings.EqualFold(session.AccessScope, "own") && !strings.EqualFold(session.AccessScope, "any") {
		helpers.WriteError(w, 403, "forbidden", "Profile access denied")
		return 0
	}
	return session.UserID
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := h.actor(w, r)
	if id == 0 {
		return
	}
	v, err := h.store.Get(r.Context(), id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, v)
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := h.actor(w, r)
	if id == 0 {
		return
	}
	var input ProfileUpdateRequest
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid profile")
		return
	}
	if err := (service.Service{Store: h.store}).Save(r.Context(), id, input); err != nil {
		h.failure(w, err)
		return
	}
	helpers.WriteJSON(w, 200, map[string]string{"status": "ok"})
}
func (h *Handler) RequestEmail(w http.ResponseWriter, r *http.Request) {
	id := h.actor(w, r)
	if id == 0 {
		return
	}
	var input models.EmailChangeRequest
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid email request")
		return
	}
	if err := (service.Service{Store: h.store, Mailer: h.appConf.EmailService()}).RequestEmail(r.Context(), id, input.Email); err != nil {
		h.failure(w, err)
		return
	}
	helpers.WriteJSON(w, 202, map[string]string{"status": "verification_required"})
}
func (h *Handler) ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	id := h.actor(w, r)
	if id == 0 {
		return
	}
	var input models.EmailChangeProof
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid verification")
		return
	}
	if err := (service.Service{Store: h.store}).ConfirmEmail(r.Context(), id, input); err != nil {
		h.failure(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/", Domain: h.appConf.Settings.Host, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	helpers.WriteJSON(w, 200, map[string]bool{"reauthentication_required": true})
}
func (h *Handler) failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		helpers.WriteError(w, 422, "invalid_profile", "Check profile values and age group")
	case errors.Is(err, service.ErrEmailChange):
		helpers.WriteError(w, 409, "email_verification_required", "Use the verified email-change flow")
	case errors.Is(err, service.ErrProof):
		helpers.WriteError(w, 400, "invalid_verification", "Verification is invalid or expired")
	case errors.Is(err, service.ErrBusy):
		helpers.WriteError(w, 429, "rate_limited", "Wait before requesting new codes")
	case errors.Is(err, service.ErrMail):
		helpers.WriteError(w, 503, "delivery_unavailable", "Verification delivery is unavailable")
	default:
		helpers.WriteDataError(w, err)
	}
}
