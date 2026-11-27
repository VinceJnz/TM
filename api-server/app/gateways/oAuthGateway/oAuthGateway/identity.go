package oAuthGateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

var (
	ErrExchange        = errors.New("OAuth exchange failed")
	ErrIdentityRequest = errors.New("OAuth identity request failed")
	ErrIdentityFetch   = errors.New("OAuth identity fetch failed")
	ErrInvalidIdentity = errors.New("verified provider identity required")
)

type Identity struct {
	Sub      string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Verified bool   `json:"email_verified"`
}

func (g *Gateway) FetchIdentity(ctx context.Context, userInfoURL, code, verifier string) (Identity, error) {
	token, err := g.OAuthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Identity{}, ErrExchange
	}
	client := g.OAuthConfig.Client(ctx, token)
	client.Timeout = 15 * time.Second
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return Identity{}, ErrIdentityRequest
	}
	resp, err := client.Do(request)
	if err != nil {
		return Identity{}, ErrIdentityFetch
	}
	defer resp.Body.Close()
	var identity Identity
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&identity) != nil || identity.Sub == "" || !identity.Verified {
		return Identity{}, ErrInvalidIdentity
	}
	return identity, nil
}
