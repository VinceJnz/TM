package authentication

import (
	"context"
	"github.com/jmoiron/sqlx"
	"time"
)

func (s Store) DiscardAnonymousProof(ctx context.Context, name, raw string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM st_token WHERE name=$1 AND token=$2 AND user_id=0`, name, Digest(raw))
	return err
}

func (s Store) Issue(ctx context.Context, tx *sqlx.Tx, id int, host, name, raw string, expires time.Time, data string) error {
	var value any
	if data != "" {
		value = data
	}
	var tokenID int
	if err := tx.GetContext(ctx, &tokenID, `INSERT INTO st_token (user_id,name,host,token,session_data,token_valid,valid_from,valid_to) VALUES($1,$2,$3,$4,$5,true,CURRENT_TIMESTAMP,$6) RETURNING id`, id, name, host, Digest(raw), value, expires); err != nil {
		return err
	}
	if id > 0 {
		_, err := tx.ExecContext(ctx, `DELETE FROM st_token WHERE user_id=$1 AND name=$2 AND substring(host FROM '^[^:]+')=substring($3 FROM '^[^:]+') AND id<>$4`, id, name, host, tokenID)
		return err
	}
	return nil
}
