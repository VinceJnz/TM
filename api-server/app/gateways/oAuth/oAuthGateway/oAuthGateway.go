package oAuthGateway

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Gateway holds shared OAuth config and the session store used by handlers.
type Gateway struct {
	OAuthConfig    *oauth2.Config
	Store          *sessions.CookieStore
	ClientRedirect string // where to redirect users after login/logout
	SessionName    string // name of the cookie session
}

type settings struct {
	// Required env:
	GOOGLE_CLIENT_ID     string
	GOOGLE_CLIENT_SECRET string
	GOOGLE_REDIRECT_URL  string

	// Optional env:
	SESSION_KEY         string
	CLIENT_REDIRECT_URL string
	SESSION_NAME        string
	SESSION_MAXAGE      int
	DEV                 bool
}

//   - CLIENT_REDIRECT_URL (default "http://localhost:8081")
//   - SESSION_NAME (default "auth-session")
//   - SESSION_MAXAGE (seconds, default 604800 = 7d)
//   - DEV (if set to "1" then cookie Secure=false to ease local dev)

// add any specific settings needed for the OAuth gateway here

// NewFromEnv constructs a Gateway configured from environment variables.
//
// Required env:
//   - GOOGLE_CLIENT_ID
//   - GOOGLE_CLIENT_SECRET
//   - GOOGLE_REDIRECT_URL
//
// Optional env:
//   - SESSION_KEY (32+ bytes recommended)
//   - CLIENT_REDIRECT_URL (default "http://localhost:8081")
//   - SESSION_NAME (default "auth-session")
//   - SESSION_MAXAGE (seconds, default 604800 = 7d)
//   - DEV (if set to "1" then cookie Secure=false to ease local dev)
func NewFromEnv() (*Gateway, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL must be set")
	}

	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		log.Println("WARNING: SESSION_KEY not set; using insecure fallback (dev only). Set SESSION_KEY in production.")
		sessionKey = "insecure-dev-session-key-please-set_SESSION_KEY"
	}

	clientRedirect := os.Getenv("CLIENT_REDIRECT_URL")
	if clientRedirect == "" {
		clientRedirect = "http://localhost:8081"
	}

	sessionName := os.Getenv("SESSION_NAME")
	if sessionName == "" {
		sessionName = "auth-session"
	}

	maxAge := 86400 * 7 // 7 days
	if v := os.Getenv("SESSION_MAXAGE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			maxAge = i
		}
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}

	store := sessions.NewCookieStore([]byte(sessionKey))
	dev := os.Getenv("DEV") == "1"
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !dev, // allow non-TLS in local dev when DEV=1
		SameSite: http.SameSiteLaxMode,
	}

	return &Gateway{
		OAuthConfig:    cfg,
		Store:          store,
		ClientRedirect: clientRedirect,
		SessionName:    sessionName,
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

// AuthCodeURL returns the AuthCodeURL for the given state. If offline is true
// the request will ask for a refresh token (AccessTypeOffline + prompt=consent).
func (g *Gateway) AuthCodeURL(state string, offline bool) string {
	if offline {
		return g.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	}
	return g.OAuthConfig.AuthCodeURL(state)
}

// GetSession convenience wrapper to fetch the gorilla session.
func (g *Gateway) GetSession(r *http.Request) (*sessions.Session, error) {
	return g.Store.Get(r, g.SessionName)
}

// SaveSession convenience wrapper to save the session and report any error.
func (g *Gateway) SaveSession(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	// keep a defensive short deadline for session persistence where appropriate
	// (gorilla sessions writes cookie to ResponseWriter only)
	return s.Save(r, w)
}

// ClearSession deletes the session cookie.
func (g *Gateway) ClearSession(r *http.Request, w http.ResponseWriter) {
	s, err := g.GetSession(r)
	if err == nil {
		s.Options.MaxAge = -1
		_ = s.Save(r, w)
	}
}

// Recommended helper: create a fresh state token, persist to session and return it.
func (g *Gateway) NewStateToken(r *http.Request, w http.ResponseWriter) (string, error) {
	s, err := g.GetSession(r)
	if err != nil {
		return "", err
	}
	state, err := RandString(32)
	if err != nil {
		return "", err
	}
	s.Values["oauth-state"] = state
	if err := g.SaveSession(r, w, s); err != nil {
		return "", err
	}
	// rotate expiry on token storage ???????????? what needs to be done here?
	//s.Options.MaxAge = s.Options.MaxAge // keep same
	//s.Options.MaxAge = s.Options.MaxAge // keep same
	return state, nil
}

// ValidateAndClearState validates state from the request vs session and clears it.
func (g *Gateway) ValidateAndClearState(r *http.Request, w http.ResponseWriter, state string) (bool, error) {
	s, err := g.GetSession(r)
	if err != nil {
		return false, err
	}
	stored, _ := s.Values["oauth-state"].(string)
	if stored == "" || state == "" || stored != state {
		return false, nil
	}
	delete(s.Values, "oauth-state")
	if err := g.SaveSession(r, w, s); err != nil {
		return false, err
	}
	return true, nil
}

// ...existing code...
