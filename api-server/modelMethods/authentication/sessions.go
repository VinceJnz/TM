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

func (s Store) RevokeSession(ctx context.Context, raw string, userID int) error {
	var id int
	return s.DB.GetContext(ctx, &id, `DELETE FROM st_token t USING st_users u
		WHERE t.user_id=u.id AND u.id=$1 AND u.user_account_status_id=$2
			AND t.name='session' AND t.token=$3 AND t.token_valid
			AND t.valid_from<CURRENT_TIMESTAMP AND t.valid_to>CURRENT_TIMESTAMP
		RETURNING t.id`, userID, models.AccountActive, Digest(raw))
}
