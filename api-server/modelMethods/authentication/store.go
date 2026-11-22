// Package authentication owns SQL for authentication commands, without HTTP types.
package authentication

import (
	"api-server/v2/models"
	"context"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }

// LinkIdentity updates only identity columns. The predicate is checked under the
// UPDATE lock, so an intervening disable/email change cannot be overwritten.
func (s Store) LinkIdentity(ctx context.Context, id int, provider, subject, email string) (int, error) {
	var linked int
	err := s.DB.GetContext(ctx, &linked, `UPDATE st_users SET provider=$2,provider_id=$3
 WHERE id=$1 AND lower(email)=lower($4) AND user_account_status_id=$5
 AND (COALESCE(provider,'')='' OR (provider=$2 AND (COALESCE(provider_id,'')='' OR provider_id=$3))) RETURNING id`, id, provider, subject, email, models.AccountActive)
	return linked, err
}
