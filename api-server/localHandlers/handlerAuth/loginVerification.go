package handlerAuth

import (
	auth "api-server/v2/app/services/authentication"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/authentication"
	"errors"
	"net/http"
)

func (h *Handler) verifyLoginOTP(w http.ResponseWriter, r *http.Request, name string) {
	var input struct {
		Token      string `json:"token"`
		RememberMe bool   `json:"remember_me"`
	}
	if helpers.DecodeJSONBody(r, &input) != nil || input.Token == "" {
		helpers.WriteError(w, 400, "invalid_request", "OTP token is required")
		return
	}
	expires := helpers.SessionExpiryFromRememberMe(input.RememberMe)
	raw, user, err := (auth.Service{Store: store.Store{DB: h.appConf.Db}}).Login(r.Context(), name, input.Token, h.appConf.Settings.Host, expires)
	if err != nil {
		if errors.Is(err, auth.ErrToken) || errors.Is(err, auth.ErrInactive) {
			helpers.WriteError(w, 403, "invalid_login", "Invalid or expired OTP, or account is not active")
		} else {
			helpers.WriteError(w, 500, "login_failed", "Failed to complete login; please retry")
		}
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: raw, Path: "/", Domain: h.appConf.Settings.Host, Expires: expires, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	helpers.WriteJSON(w, 200, map[string]any{"status": "logged_in", "user_id": user.ID, "username": user.Username, "email": user.Email.String, "name": user.Name})
}
