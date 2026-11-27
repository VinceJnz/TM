package authentication

import (
	store "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
)

func (s Service) ConsumeOAuthState(ctx context.Context, raw string) (models.Token, error) {
	return store.Consume(ctx, s.Store.DB, "oauth-state", raw)
}

func (s Service) DiscardOAuthRegistration(ctx context.Context, raw string) error {
	_, err := store.Consume(ctx, s.Store.DB, "oauth-registration", raw)
	return err
}
