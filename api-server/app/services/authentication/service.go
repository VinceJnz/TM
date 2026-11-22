package authentication

import (
	store "api-server/v2/modelMethods/authentication"
	"context"
	"errors"
	"strings"
)

var ErrIdentity = errors.New("verified identity required")

type Service struct{ Store store.Store }

// Only the gateway-verified provider identity may be supplied here.
func (s Service) LinkIdentity(ctx context.Context, id int, provider, subject, email string, verified bool) (int, error) {
	if !verified || id <= 0 || provider != "google" || strings.TrimSpace(subject) == "" || strings.TrimSpace(email) == "" {
		return 0, ErrIdentity
	}
	return s.Store.LinkIdentity(ctx, id, provider, subject, email)
}
