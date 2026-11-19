package dbAuthTemplate

import (
	"api-server/v2/models"
	"context"

	"github.com/jmoiron/sqlx"
)

// ConsumeToken lets exactly one caller use a valid one-time token, including
// across API replicas. A replay cannot succeed between a lookup and deletion.
func ConsumeToken(ctx context.Context, db *sqlx.DB, name, value string) (models.Token, error) {
	var token models.Token
	err := db.QueryRowContext(ctx, `DELETE FROM st_token WHERE name=$1 AND token=$2 AND token_valid AND valid_from<CURRENT_TIMESTAMP AND valid_to>CURRENT_TIMESTAMP RETURNING id,user_id,name,token,session_data,token_valid,valid_from,valid_to`, name, value).Scan(
		&token.ID, &token.UserID, &token.Name, &token.TokenStr, &token.SessionData, &token.Valid, &token.ValidFrom, &token.ValidTo)
	return token, err
}
