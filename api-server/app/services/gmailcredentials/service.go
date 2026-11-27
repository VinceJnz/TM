package gmailcredentials

import (
	"api-server/v2/app/gateways/gmail"
	"api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type Config struct {
	TokenFile, SecretFile, Email, RecipientOverride string
	Redirect, CallbackPath                          string
	DevMode                                         bool
	Activate                                        func(*gmail.Gateway)
	Loaded                                          func() bool
}

type Service struct {
	config  Config
	mu      sync.Mutex
	pending renewal
}

func New(config Config) *Service {
	return &Service{config: config}
}

type renewal struct {
	state, verifier string
	session         [32]byte
	expires         time.Time
	config          *oauth2.Config
}

type ErrorKind int

const (
	Invalid ErrorKind = iota
	Conflict
	Unavailable
	ProviderFailure
	Internal
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

type Status struct {
	RefreshAvailable bool       `json:"gmail_refresh_available"`
	TokenExpiry      *time.Time `json:"gmail_token_expiry"`
	Email            string     `json:"email_addr"`
	TokenFile        string     `json:"token_file"`
	TokenExists      bool       `json:"token_exists"`
	TokenModified    time.Time  `json:"token_modified"`
	SecretFile       string     `json:"secret_file"`
	SecretExists     bool       `json:"secret_exists"`
	SecretModified   time.Time  `json:"secret_modified"`
	Loaded           bool       `json:"gmail_loaded"`
	CanReconnect     bool       `json:"can_reconnect"`
}

func (s *Service) Status() Status {
	c := s.config
	tokenExists, tokenModified := fileInfo(c.TokenFile)
	secretExists, secretModified := fileInfo(c.SecretFile)
	token, err := readToken(c.TokenFile)
	var expiry *time.Time
	if err == nil && !token.Expiry.IsZero() {
		expiry = &token.Expiry
	}
	return Status{
		RefreshAvailable: secretExists && err == nil && token.RefreshToken != "",
		TokenExpiry:      expiry, Email: c.Email,
		TokenFile: c.TokenFile, TokenExists: tokenExists, TokenModified: tokenModified,
		SecretFile: c.SecretFile, SecretExists: secretExists, SecretModified: secretModified,
		Loaded: c.Loaded(), CanReconnect: secretExists && c.TokenFile != "" && s.validRedirect(),
	}
}

func fileInfo(path string) (bool, time.Time) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false, time.Time{}
	}
	return true, info.ModTime()
}

func readToken(path string) (oauth2.Token, error) {
	var token oauth2.Token
	content, err := os.ReadFile(path)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal(content, &token)
	return token, err
}

func (s *Service) validRedirect() bool {
	c := s.config
	u, err := url.Parse(c.Redirect)
	return err == nil && u.Hostname() != "" && u.User == nil && u.RawQuery == "" &&
		u.Fragment == "" && u.Path == c.CallbackPath &&
		(u.Scheme == "https" || (c.DevMode && u.Scheme == "http" &&
			(u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")))
}

func (s *Service) oauthConfig() (*oauth2.Config, error) {
	content, err := os.ReadFile(s.config.SecretFile)
	if err != nil {
		return nil, err
	}
	return gmail.OAuthConfig(content)
}

func (s *Service) BeginRenewal(session [32]byte) (string, error) {
	if !s.validRedirect() || s.config.TokenFile == "" {
		return "", &Error{Unavailable, "Configure GMAIL_RENEW_REDIRECT_URL and EMAIL_TOKEN to reconnect Gmail"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	config, err := s.oauthConfig()
	if err != nil {
		return "", &Error{Conflict, "Upload valid Gmail OAuth client credentials first"}
	}
	state, err := oAuthGateway.RandString(32)
	if err != nil {
		return "", &Error{Internal, "Could not start Gmail authorization"}
	}
	verifier := oauth2.GenerateVerifier()
	config.RedirectURL = s.config.Redirect
	s.pending = renewal{state: state, verifier: verifier, session: session, expires: time.Now().Add(10 * time.Minute), config: config}
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("login_hint", s.config.Email)), nil
}

// takeRenewal is called with the service lock held.
func (s *Service) takeRenewal(state string, session [32]byte, now time.Time) (renewal, bool) {
	pending := s.pending
	if state == "" || state != pending.state || session != pending.session || !now.Before(pending.expires) {
		return renewal{}, false
	}
	s.pending = renewal{}
	return pending, true
}

func (s *Service) CompleteRenewal(ctx context.Context, session [32]byte, state, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, ok := s.takeRenewal(state, session, time.Now())
	if !ok {
		return &Error{Invalid, "Gmail authorization expired. Return to Gateway Credentials and reconnect again."}
	}
	if code == "" {
		return &Error{Invalid, "Google did not authorize Gmail access"}
	}
	token, err := gmail.ExchangeCredential(ctx, pending.config, code, pending.verifier)
	if err != nil || token.RefreshToken == "" || !token.Valid() {
		return &Error{ProviderFailure, "Google did not provide a usable Gmail token. Reconnect and approve offline access."}
	}
	if err := s.installToken(pending.config, token); err != nil {
		return &Error{Internal, "Could not save the Gmail token. Check the configured token file permissions."}
	}
	return nil
}

func (s *Service) Refresh(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, err := s.oauthConfig()
	if err != nil {
		return &Error{Conflict, "Gmail OAuth client credentials are unavailable"}
	}
	token, err := readToken(s.config.TokenFile)
	if err != nil || token.RefreshToken == "" {
		return &Error{Conflict, "Gmail has no saved refresh token. Reconnect Gmail."}
	}
	refreshed, err := gmail.RefreshCredential(ctx, config, token)
	if err != nil || !refreshed.Valid() {
		return &Error{ProviderFailure, "Gmail token refresh failed. Retry or reconnect Gmail if access was revoked."}
	}
	if err := s.installToken(config, refreshed); err != nil {
		return &Error{Internal, "Could not save the refreshed Gmail token. Check the configured token file permissions."}
	}
	s.pending = renewal{}
	return nil
}

func (s *Service) Update(kind string, content []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var message string
	switch kind {
	case "secret":
		if _, err := gmail.OAuthConfig(content); err != nil {
			return "", &Error{Invalid, "Invalid Google OAuth client credentials"}
		}
		if err := writeCredential(s.config.SecretFile, content); err != nil {
			return "", &Error{Internal, "Could not save Gmail client credentials"}
		}
		message = "Gmail client credentials saved. Refresh or reconnect Gmail to activate them."
	case "token":
		var token oauth2.Token
		if json.Unmarshal(content, &token) != nil || token.RefreshToken == "" {
			return "", &Error{Invalid, "A Gmail token with a refresh token is required"}
		}
		config, err := s.oauthConfig()
		if err != nil {
			return "", &Error{Conflict, "Upload valid Gmail OAuth client credentials first"}
		}
		if err := s.installToken(config, &token); err != nil {
			return "", &Error{Internal, "Could not install the Gmail token"}
		}
		message = "Gmail token installed."
	default:
		return "", &Error{Invalid, "Credential type must be token or secret"}
	}
	s.pending = renewal{}
	return message, nil
}

func (s *Service) installToken(config *oauth2.Config, token *oauth2.Token) error {
	gateway, err := gmail.NewWithToken(config, token, s.config.Email, s.config.RecipientOverride)
	if err != nil {
		return err
	}
	content, err := json.Marshal(token)
	if err != nil {
		return err
	}
	if err := writeCredential(s.config.TokenFile, content); err != nil {
		return err
	}
	s.config.Activate(gateway)
	return nil
}

func writeCredential(path string, content []byte) error {
	if path == "" {
		return errors.New("credential path is missing")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".gmail-credential-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
