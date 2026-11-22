package dbAuthTemplate

import (
	store "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
	"github.com/jmoiron/sqlx"
)

func ConsumeToken(ctx context.Context, db *sqlx.DB, name, value string) (models.Token, error) {
	return store.Consume(ctx, db, name, value)
}
