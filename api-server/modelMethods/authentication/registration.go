package authentication

import (
	"api-server/v2/models"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/jmoiron/sqlx"
)

func Digest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func Consume(ctx context.Context, q sqlx.QueryerContext, name, value string) (models.Token, error) {
	var t models.Token
	err := q.QueryRowxContext(ctx, `DELETE FROM st_token WHERE name=$1 AND token=$2 AND token_valid AND valid_from<CURRENT_TIMESTAMP AND valid_to>CURRENT_TIMESTAMP RETURNING id,user_id,name,token,session_data,token_valid,valid_from,valid_to`, name, Digest(value)).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenStr, &t.SessionData, &t.Valid, &t.ValidFrom, &t.ValidTo)
	return t, err
}
func (s Store) InsertUser(ctx context.Context, tx *sqlx.Tx, u models.User) (int, error) {
	var id int
	err := tx.GetContext(ctx, &id, `INSERT INTO st_users (name, username, email, user_address, member_code, user_birth_date, user_age_group_id, user_password, user_account_status_id, user_account_hidden, provider, provider_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`, u.Name, u.Username, u.Email, u.Address, u.MemberCode, u.BirthDate, u.UserAgeGroupID, u.Password, u.AccountStatusID, u.AccountHidden, u.Provider, u.ProviderID)
	return id, err
}
func (s Store) AgeGroup(ctx context.Context, tx *sqlx.Tx, id int64) (string, error) {
	var label string
	var query sqlx.QueryerContext = s.DB
	if tx != nil {
		query = tx
	}
	err := sqlx.GetContext(ctx, query, &label, `SELECT age_group FROM et_user_age_groups WHERE id=$1`, id)
	return label, err
}
