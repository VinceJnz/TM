package dbAuthTemplate

import (
	auth "api-server/v2/modelMethods/authentication"
	"context"

	"github.com/jmoiron/sqlx"
)

func CreatePasswordReset(ctx context.Context, db *sqlx.DB, email string) (string, string, error) {
	code := GenerateSecureToken()
	recipient, err := (auth.Store{DB: db}).CreatePasswordReset(ctx, email, code)
	if err != nil {
		return "", "", err
	}
	return recipient, code, nil
}

func ResetPassword(ctx context.Context, db *sqlx.DB, code, hash string) error {
	return (auth.Store{DB: db}).ResetPassword(ctx, code, hash)
}
