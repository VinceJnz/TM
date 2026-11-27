package handlerAuth

import (
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"net/http"
	"net/url"
	"time"
)

func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Use POST to log out", 405)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme != "https" || parsed.Host != r.Host || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			http.Error(w, "Origin not allowed", 403)
			return
		}
	}
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session)
	if !ok || session == nil {
		http.Error(w, "Authentication required", 401)
		return
	}
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Authentication required", 401)
		return
	}
	if err = h.authentication().Logout(r.Context(), cookie.Value, session.UserID); err != nil {
		http.Error(w, "Unable to close session", 503)
		return
	}
	helpers.SetSessionCookie(w, r, h.appConf.Settings.Host, "", time.Time{})
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("User logged out."))
}
