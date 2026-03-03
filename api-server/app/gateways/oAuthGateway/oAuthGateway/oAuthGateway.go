package oAuthGateway

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"

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

type GatewayConfig struct {
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	SessionKey     string
	ClientRedirect string
	DevMode        bool
}

// New builds a Gateway from app-provided settings.
func New(cfg GatewayConfig) (*Gateway, error) {
	clientID := cfg.ClientID
	clientSecret := cfg.ClientSecret
	redirectURL := cfg.RedirectURL
	sessionKey := cfg.SessionKey
	devMode := cfg.DevMode
	clientRedirect := cfg.ClientRedirect
	if clientRedirect == "" {
		clientRedirect = "http://localhost:8081"
	}

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL must be set")
	}
	if sessionKey == "" {
		if !devMode {
			return nil, errors.New("SESSION_KEY must be set when DEV mode is not enabled")
		}
		generatedKey, err := RandString(32)
		if err != nil {
			return nil, errors.New("failed to generate DEV session key")
		}
		log.Println("WARNING: SESSION_KEY not set; using generated ephemeral key because DEV mode is enabled")
		sessionKey = generatedKey
	}

	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}

	store := sessions.NewCookieStore([]byte(sessionKey))
	secure := !devMode // if DEV mode is enabled, keep Secure=false for local dev
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   secure,
		SameSite: 1, // http.SameSiteLaxMode as int literal to avoid extra import
	}

	return &Gateway{
		OAuthConfig:    oauthCfg,
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
