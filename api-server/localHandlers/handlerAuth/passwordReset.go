package handlerAuth

import (
	handlerHelpers "api-server/v2/localHandlers/helpers"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

var passwordResetWorkers = make(chan struct{}, 8)

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
	debugRecipient := h.appConf.Settings.EmailRecipientOverride()
	// Only the debug inbox itself and the explicitly configured test account
	// may use the debug inbox for password resets.
	if mailer == nil || (debugRecipient != "" && !strings.EqualFold(debugRecipient, email) &&
		!strings.EqualFold(h.appConf.Settings.PasswordResetTestEmail, email)) {
		handlerHelpers.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "Password reset email is unavailable.")
		return
	}

	// Keep account lookup and email delivery out of the response timing.
	select {
	case passwordResetWorkers <- struct{}{}:
	default:
		handlerHelpers.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "Password reset email is busy. Try again shortly.")
		return
	}
	go func(ctx context.Context) {
		defer func() { <-passwordResetWorkers }()
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		recipient, code, err := h.authentication().CreatePasswordReset(ctx, email)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				log.Print("password reset code creation failed")
			}
			return
		}
		if debugRecipient != "" {
			recipient = debugRecipient
		}
		const subject = "Reset your Trip Manager password"
		message := "Enter this code in the Forgot password form. It expires in 15 minutes.\n\n" + code +
			"\n\nIf you did not request a reset, you can ignore this message."
		var sent bool
		if contextual, ok := mailer.(interface {
			SendMailContext(context.Context, string, string, string) (bool, error)
		}); ok {
			sent, err = contextual.SendMailContext(ctx, recipient, subject, message)
		} else {
			sent, err = mailer.SendMail(recipient, subject, message)
		}
		if err != nil || !sent {
			log.Print("password reset email delivery failed")
		}
	}(context.WithoutCancel(r.Context()))
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
