package helpers

import (
	authService "api-server/v2/app/services/authentication"
	authStore "api-server/v2/modelMethods/authentication"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

func CreateAndSetSessionCookie(r *http.Request, debugStr string, w http.ResponseWriter, db *sqlx.DB, userID int, host string, expiration time.Time) error {
	auth := authService.Service{Store: authStore.Store{DB: db}}
	raw, err := auth.Issue(r.Context(), userID, host, "session", expiration, "")
	if err != nil {
		log.Printf("%vCreateAndSetSessionCookie() failed to create session token: %v", debugStr, err)
		return err
	}
	SetSessionCookie(w, r, host, raw, expiration)
	return nil
}

// SetSessionCookie scopes the session to the host that served the response.
// An empty token expires the browser session.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, legacyDomain, token string, expiration time.Time) {
	cookie := http.Cookie{Name: "session", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	host := strings.ToLower((&url.URL{Host: r.Host}).Hostname())
	domain := strings.TrimPrefix(strings.ToLower(legacyDomain), ".")
	if domain != "" && (host == domain || strings.HasSuffix(host, "."+domain)) {
		legacy := cookie
		legacy.Domain = domain
		legacy.MaxAge = -1
		legacy.Expires = time.Unix(1, 0)
		http.SetCookie(w, &legacy)
	}
	cookie.Value = token
	cookie.Expires = expiration
	if token == "" {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, &cookie)
}

func WriteAcceptedText(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(msg))
}

func WriteUnauthorizedText(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusUnauthorized)
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Printf("helpers.WriteUnauthorizedText() failed to write response: %v", err)
	}
}

func WriteForbidden(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusForbidden)
}

func WriteInternalServerError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusInternalServerError)
}

func WriteAcceptedJSON(w http.ResponseWriter, payload any) {
	WriteJSON(w, http.StatusAccepted, payload)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("helpers.WriteJSON() failed to encode response: %v", err)
	}
}

func DecodeJSONBody(r *http.Request, payload any) error {
	const maxBody = 64 << 10
	if r.Body == nil {
		return errors.New("missing request body")
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return err
	}
	if len(data) > maxBody {
		return errors.New("request body too large")
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("request object required")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err = decoder.Decode(payload); err != nil {
		return err
	}
	var trailing any
	if err = decoder.Decode(&trailing); err != io.EOF {
		return errors.New("exactly one JSON value required")
	}
	return nil
}

func SessionExpiryFromRememberMe(rememberMe bool) time.Time {
	if rememberMe {
		return time.Now().Add(30 * 24 * time.Hour)
	}

	return time.Time{}
}
