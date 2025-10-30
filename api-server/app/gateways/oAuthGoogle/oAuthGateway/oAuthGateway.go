package oAuthGateway

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Gateway holds shared OAuth config and the session store.
type Gateway struct {
	OAuthConfig    *oauth2.Config
	Store          *sessions.CookieStore
	ClientRedirect string
}

func New() (*Gateway, error) {
	return NewFromEnv()
}

// NewFromEnv builds a Gateway from environment variables.
// Required: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL
// Optional: SESSION_KEY, CLIENT_REDIRECT_URL, DEV (if set, cookie Secure=false)
func NewFromEnv() (*Gateway, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	sessionKey := os.Getenv("SESSION_KEY")
	clientRedirect := os.Getenv("CLIENT_REDIRECT_URL")
	if clientRedirect == "" {
		clientRedirect = "http://localhost:8081"
	}

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL must be set")
	}
	if sessionKey == "" {
		log.Println("WARNING: SESSION_KEY not set; using insecure fallback (dev only)")
		sessionKey = "insecure-dev-session-key-please-set_SESSION_KEY"
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}

	store := sessions.NewCookieStore([]byte(sessionKey))
	secure := os.Getenv("DEV") == "" // if DEV set, keep Secure=false for local dev
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   secure,
		SameSite: 1, // http.SameSiteLaxMode as int literal to avoid extra import
	}

	return &Gateway{
		OAuthConfig:    cfg,
		Store:          store,
		ClientRedirect: clientRedirect,
	}, nil
}

// RandString returns a URL-safe base64 string representing n random bytes.
func RandString(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("n must be > 0")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
