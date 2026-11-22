package accounts

import (
	"api-server/v2/modelMethods/resourcequery"
	security "api-server/v2/modelMethods/security"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }

func (s Store) Lookup(ctx context.Context, query string, id int) ([]resourcequery.UserOption, error) {
	return resourcequery.UserOptions(ctx, s.DB, query, id)
}

var ErrReferenced = errors.New("account has dependent records")

const columns = `version,id,name,username,COALESCE(email,'') AS email,COALESCE(user_address,'') AS user_address,COALESCE(member_code,'') AS member_code,COALESCE(user_birth_date,'0001-01-01'::date) AS user_birth_date,COALESCE(user_age_group_id,0) AS user_age_group_id,COALESCE(member_status_id,0) AS member_status_id,COALESCE(user_account_status_id,0) AS user_account_status_id,COALESCE(user_account_hidden,false) AS user_account_hidden,created,modified`

func (s Store) List(ctx context.Context, p resourcequery.Page) ([]models.AccountDTO, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[models.AccountDTO](ctx, s.DB, p, "SELECT "+columns+" FROM st_users")
}
func (s Store) Get(ctx context.Context, id int) (models.AccountDTO, error) {
	var v models.AccountDTO
	err := s.DB.GetContext(ctx, &v, "SELECT "+columns+" FROM st_users WHERE id=$1", id)
	return v, err
}
func (s Store) Begin(ctx context.Context, actor int, reason string) (*sqlx.Tx, error) {
	tx, err := (security.Store{DB: s.DB}).Begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "SELECT set_config('tm.change_reason',$1,true)", reason); err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}
func (s Store) Lock(ctx context.Context, tx *sqlx.Tx, id int) (models.AccountDTO, error) {
	var v models.AccountDTO
	err := tx.GetContext(ctx, &v, "SELECT "+columns+" FROM st_users WHERE id=$1 FOR UPDATE", id)
	return v, err
}
func (s Store) Save(ctx context.Context, tx *sqlx.Tx, id int, v models.AccountWrite) (models.AccountDTO, error) {
	args := []any{v.Name, v.Username, v.Email, v.Address, v.MemberCode, v.BirthDate, v.UserAgeGroupID, v.MemberStatusID, v.AccountStatusID, v.AccountHidden}
	q := `INSERT INTO st_users(name,username,email,user_address,member_code,user_birth_date,user_age_group_id,member_status_id,user_account_status_id,user_account_hidden) VALUES($1,$2,$3,$4,$5,NULLIF($6::date,'0001-01-01'::date),NULLIF($7,0), $8, $9,$10) RETURNING ` + columns
	if id > 0 {
		q = `UPDATE st_users SET name=$1,username=$2,email=$3,user_address=$4,member_code=$5,user_birth_date=NULLIF($6::date,'0001-01-01'::date),user_age_group_id=NULLIF($7,0),member_status_id= $8,user_account_status_id= $9,user_account_hidden=$10,modified=CURRENT_TIMESTAMP WHERE id=$11 RETURNING ` + columns
		args = append(args, id)
	}
	var result models.AccountDTO
	err := tx.GetContext(ctx, &result, q, args...)
	return result, err
}
func (s Store) Revoke(ctx context.Context, tx *sqlx.Tx, id int) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM st_email_changes WHERE user_id=$1", id); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "DELETE FROM st_token WHERE user_id=$1", id)
	return err
}
func (s Store) Delete(ctx context.Context, tx *sqlx.Tx, id int) error {
	var referenced bool
	err := tx.GetContext(ctx, &referenced, `SELECT EXISTS(SELECT 1 FROM st_user_group WHERE user_id=$1) OR EXISTS(SELECT 1 FROM at_bookings WHERE owner_id=$1) OR EXISTS(SELECT 1 FROM at_booking_people WHERE person_id=$1 OR owner_id=$1) OR EXISTS(SELECT 1 FROM at_trips WHERE owner_id=$1) OR EXISTS(SELECT 1 FROM at_group_bookings WHERE owner_id=$1)`, id)
	if err != nil {
		return err
	}
	if referenced {
		return ErrReferenced
	}
	r, err := tx.ExecContext(ctx, "DELETE FROM st_users WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
