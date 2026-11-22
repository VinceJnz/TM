package authentication

import (
	"api-server/v2/models"
	"context"
	"github.com/jmoiron/sqlx"
)

func (s Store) RegistrationUser(ctx context.Context, tx *sqlx.Tx, id int) (models.User, error) {
	var u models.User
	err := tx.GetContext(ctx, &u, `SELECT version,id,email,user_account_status_id FROM st_users WHERE id=$1 FOR UPDATE`, id)
	return u, err
}
func (s Store) RegistrationProfile(ctx context.Context, tx *sqlx.Tx, id int, u models.User) error {
	_, err := tx.ExecContext(ctx, `UPDATE st_users SET username=$2,user_address=$3,user_birth_date=$4,user_age_group_id=$5,user_account_hidden=$6 WHERE id=$1`, id, u.Username, u.Address, u.BirthDate, u.UserAgeGroupID, u.AccountHidden)
	return err
}
func (s Store) VerifyEmail(ctx context.Context, tx *sqlx.Tx, id int) error {
	_, err := tx.ExecContext(ctx, `UPDATE st_users SET user_account_status_id=$2 WHERE id=$1 AND user_account_status_id=$3`, id, models.AccountVerified, models.AccountNew)
	return err
}
