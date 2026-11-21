package handlerAuth

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"context"
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
	if err = h.removeSessionToken(r.Context(), cookie.Value, session.UserID); err != nil {
		http.Error(w, "Unable to close session", 503)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", Domain: h.appConf.Settings.Host, MaxAge: -1, Expires: time.Unix(1, 0), Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("User logged out."))
}
func (h *Handler) removeSessionToken(ctx context.Context, value string, user int) error {
	token, err := dbAuthTemplate.FindSessionTokenContext(ctx, debugTag, h.appConf.Db, value)
	if err != nil {
		return err
	}
	if token.UserID != user {
		return http.ErrNoCookie
	}
	_, err = h.appConf.Db.ExecContext(ctx, "DELETE FROM st_token WHERE id=$1 AND user_id=$2", token.ID, user)
	return err
}
