package handlerAuth

import (
	handlerHelpers "api-server/v2/localHandlers/helpers"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strings"
)

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email string `json:"email"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	err := handlerHelpers.DecodeJSONBody(r, &request)
	email := strings.TrimSpace(request.Email)
	if err != nil || len(email) > 254 || !handlerHelpers.IsValidEmail(email) {
		handlerHelpers.WriteError(w, http.StatusBadRequest, "invalid_email", "Enter a valid email address.")
		return
	}

	if h.appConf == nil || h.appConf.Db == nil {
		handlerHelpers.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "Password reset is temporarily unavailable.")
		return
	}
	mailer := h.appConf.EmailService()
	if mailer == nil {
		handlerHelpers.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "Password reset email is unavailable.")
		return
	}

	if !h.authentication().QueuePasswordReset(r.Context(), mailer, email) {
		handlerHelpers.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "Password reset email is busy. Try again shortly.")
		return
	}
	handlerHelpers.WriteAcceptedJSON(w, map[string]string{
		"message": "If this address belongs to an eligible account, a reset code will be sent.",
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := handlerHelpers.DecodeJSONBody(r, &request); err != nil {
		handlerHelpers.WriteError(w, http.StatusBadRequest, "invalid_request", "Enter the reset code and a new password.")
		return
	}
	if len(request.Password) < 8 || len(request.Password) > 72 {
		handlerHelpers.WriteError(w, http.StatusBadRequest, "invalid_password", "Password must be between 8 and 72 bytes.")
		return
	}
	code := strings.TrimSpace(request.Code)
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(code)
	if err != nil || len(decoded) != 32 {
		handlerHelpers.WriteError(w, http.StatusBadRequest, "invalid_code", "The reset code is invalid or expired.")
		return
	}
	if h.appConf == nil || h.appConf.Db == nil {
		handlerHelpers.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "Password reset is temporarily unavailable.")
		return
	}

	err = h.authentication().ResetPassword(r.Context(), code, request.Password)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		handlerHelpers.WriteError(w, http.StatusBadRequest, "invalid_code", "The reset code is invalid or expired.")
	case err != nil:
		log.Print("password reset failed")
		handlerHelpers.WriteError(w, http.StatusInternalServerError, "reset_failed", "Password reset could not be completed.")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
