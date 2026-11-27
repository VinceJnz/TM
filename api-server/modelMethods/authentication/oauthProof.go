package authentication

import (
	"api-server/v2/models"
	"context"
)

func (s Store) OAuthRegistration(ctx context.Context, raw string) (models.Token, error) {
	var t models.Token
	err := s.DB.QueryRowxContext(ctx, `SELECT id,user_id,session_data FROM st_token
        WHERE name='oauth-registration' AND token=$1 AND token_valid
        AND valid_from<CURRENT_TIMESTAMP AND valid_to>CURRENT_TIMESTAMP`, Digest(raw)).Scan(&t.ID, &t.UserID, &t.SessionData)
	return t, err
}
