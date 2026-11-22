package authentication

import (
	"api-server/v2/models"
	"context"
)

const userColumns = `version,id,name,username,email,user_address,user_birth_date,user_age_group_id,user_account_status_id,user_password,user_account_hidden,provider,provider_id`

func (s Store) User(ctx context.Context, id int) (models.User, error) {
	var u models.User
	err := s.DB.GetContext(ctx, &u, "SELECT "+userColumns+" FROM st_users WHERE id=$1", id)
	return u, err
}
func (s Store) Username(ctx context.Context, name string) (models.User, error) {
	var u models.User
	err := s.DB.GetContext(ctx, &u, "SELECT "+userColumns+" FROM st_users WHERE username=$1", name)
	return u, err
}
func (s Store) Email(ctx context.Context, email string) (models.User, error) {
	var u models.User
	err := s.DB.GetContext(ctx, &u, "SELECT "+userColumns+" FROM st_users WHERE email=$1", email)
	return u, err
}
func (s Store) Principal(ctx context.Context, raw string) (models.User, error) {
	var u models.User
	err := s.DB.GetContext(ctx, &u, `SELECT u.id,u.name,u.username,u.email,u.user_account_status_id FROM st_users u JOIN st_token t ON t.user_id=u.id WHERE t.token=$1 AND t.name='session' AND t.token_valid AND t.valid_from<CURRENT_TIMESTAMP AND t.valid_to>CURRENT_TIMESTAMP AND u.user_account_status_id=$2`, Digest(raw), models.AccountActive)
	return u, err
}
