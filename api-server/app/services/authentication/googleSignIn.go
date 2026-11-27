package authentication

import (
	"api-server/v2/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrOAuthLookup       = errors.New("failed to load account")
	ErrOAuthLink         = errors.New("failed to link account")
	ErrOAuthSession      = errors.New("failed to create session")
	ErrOAuthContinuation = errors.New("failed to prepare registration")
)

type GoogleSignIn struct {
	Session            string
	Continuation       string
	ContinuationExpiry time.Time
	NeedsRegistration  bool
}

func (s Service) SignInWithGoogle(ctx context.Context, host, subject, email, name string, verified bool) (GoogleSignIn, error) {
	if !verified || subject == "" || email == "" {
		return GoogleSignIn{}, ErrIdentity
	}
	user, found, err := s.Store.Identity(ctx, "google", subject, email)
	if err != nil {
		return GoogleSignIn{}, fmt.Errorf("%w: %w", ErrOAuthLookup, err)
	}
	if found && user.UserActive() {
		id, err := s.LinkIdentity(ctx, user.ID, "google", subject, email, verified)
		if err != nil {
			return GoogleSignIn{}, fmt.Errorf("%w: %w", ErrOAuthLink, err)
		}
		raw, err := s.Issue(ctx, id, host, "session", time.Time{}, "")
		if err != nil {
			return GoogleSignIn{}, fmt.Errorf("%w: %w", ErrOAuthSession, err)
		}
		return GoogleSignIn{Session: raw, NeedsRegistration: user.Username == ""}, nil
	}
	if found && user.AccountStatusID.Int64 != int64(models.AccountNew) && user.AccountStatusID.Int64 != int64(models.AccountVerified) {
		return GoogleSignIn{}, ErrInactive
	}
	result := GoogleSignIn{NeedsRegistration: true, ContinuationExpiry: time.Now().Add(10 * time.Minute)}
	pending, err := json.Marshal(OAuthPendingRegistration{Email: email, Name: name, Provider: "google", ProviderID: subject})
	if err != nil {
		return GoogleSignIn{}, fmt.Errorf("%w: %w", ErrOAuthContinuation, err)
	}
	result.Continuation, err = s.Issue(ctx, user.ID, host, "oauth-registration", result.ContinuationExpiry, string(pending))
	if err != nil {
		return GoogleSignIn{}, fmt.Errorf("%w: %w", ErrOAuthContinuation, err)
	}
	return result, nil
}

type OAuthPendingRegistration struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
}
