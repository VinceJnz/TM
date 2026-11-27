package gmail

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

func ExchangeCredential(ctx context.Context, config *oauth2.Config, code, verifier string) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

func RefreshCredential(ctx context.Context, config *oauth2.Config, token oauth2.Token) (*oauth2.Token, error) {
	token.AccessToken = "" // Force an exchange even if the cached token is valid.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return config.TokenSource(ctx, &token).Token()
}
