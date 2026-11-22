package authentication

import (
	"api-server/v2/models"
	"context"
	"github.com/jmoiron/sqlx"
	"time"
)

func (s Store) LoginUser(ctx context.Context, tx *sqlx.Tx, id int) (models.User, error) {
	var u models.User
	err := tx.GetContext(ctx, &u, `SELECT id,name,username,email,user_account_status_id FROM st_users WHERE id=$1 FOR UPDATE`, id)
	return u, err
}
func (s Store) Session(ctx context.Context, tx *sqlx.Tx, id int, host, raw string, expires time.Time) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM st_token WHERE user_id=$1 AND name='session' AND substring(host FROM '^[^:]+')=substring($2 FROM '^[^:]+')`, id, host); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO st_token(user_id,name,host,token,session_data,token_valid,valid_from,valid_to) VALUES($1,'session',$2,$3,'{}',true,CURRENT_TIMESTAMP,$4)`, id, host, Digest(raw), expires)
	return err
}
