package profiles

import (
	"api-server/v2/internal/observability"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"strconv"
	"time"
)

type Store struct{ DB *sqlx.DB }

func (s Store) Begin(ctx context.Context, actor int) (*sqlx.Tx, error) {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "SELECT set_config('tm.actor',$1,true),set_config('tm.request_id',$2,true)", "user:"+strconv.Itoa(actor), observability.RequestID(ctx)); err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

const profileRead = `SELECT version,id,name,username,COALESCE(email,'') AS email,COALESCE(user_address,'') AS user_address,COALESCE(member_code,'') AS member_code,COALESCE(user_birth_date,'0001-01-01'::date) AS user_birth_date,COALESCE(user_account_hidden,false) AS user_account_hidden FROM st_users WHERE id=$1`

func (s Store) Get(ctx context.Context, id int) (models.ProfileDTO, error) {
	var v models.ProfileDTO
	err := s.DB.GetContext(ctx, &v, profileRead, id)
	return v, err
}
func (s Store) Lock(ctx context.Context, tx *sqlx.Tx, id int) (models.ProfileDTO, error) {
	var v models.ProfileDTO
	err := tx.GetContext(ctx, &v, profileRead+" FOR UPDATE", id)
	return v, err
}
func (s Store) Save(ctx context.Context, tx *sqlx.Tx, id int, v models.ProfileWrite, dob time.Time) error {
	_, err := tx.ExecContext(ctx, "UPDATE st_users SET name=$2,user_address=$3,user_birth_date=$4,user_account_hidden=$5 WHERE id=$1", id, v.Name, v.Address, dob, v.AccountHidden)
	return err
}
func (s Store) SetUsername(ctx context.Context, tx *sqlx.Tx, id int, username string) error {
	_, err := tx.ExecContext(ctx, "UPDATE st_users SET username=$2 WHERE id=$1", id, username)
	return err
}
func (s Store) AgeGroup(ctx context.Context, tx *sqlx.Tx, id int) (string, error) {
	var label string
	err := tx.GetContext(ctx, &label, "SELECT COALESCE(a.age_group,'') FROM st_users u LEFT JOIN et_user_age_groups a ON a.id=u.user_age_group_id WHERE u.id=$1", id)
	return label, err
}

type Pending struct {
	OldEmail  string `db:"old_email"`
	NewEmail  string `db:"new_email"`
	OldDigest []byte `db:"old_digest"`
	NewDigest []byte `db:"new_digest"`
}

func (s Store) Prepare(ctx context.Context, tx *sqlx.Tx, id int, oldEmail, newEmail string, oldDigest, newDigest []byte) error {
	var identity int
	return tx.GetContext(ctx, &identity, `INSERT INTO st_email_changes(user_id,old_email,new_email,old_digest,new_digest) VALUES($1,$2,$3,$4,$5)
 ON CONFLICT(user_id) DO UPDATE SET old_email=$2,new_email=$3,old_digest=$4,new_digest=$5,created=CURRENT_TIMESTAMP,expires_at=CURRENT_TIMESTAMP+interval '15 minutes',attempts=0
 WHERE st_email_changes.created<CURRENT_TIMESTAMP-interval '30 seconds' RETURNING user_id`, id, oldEmail, newEmail, oldDigest, newDigest)
}
func (s Store) Pending(ctx context.Context, tx *sqlx.Tx, id int) (Pending, error) {
	var v Pending
	err := tx.GetContext(ctx, &v, `SELECT old_email,new_email,old_digest,new_digest FROM st_email_changes WHERE user_id=$1 AND expires_at>CURRENT_TIMESTAMP AND attempts<5 FOR UPDATE`, id)
	return v, err
}
func (s Store) FailedProof(ctx context.Context, tx *sqlx.Tx, id int) error {
	_, err := tx.ExecContext(ctx, "UPDATE st_email_changes SET attempts=attempts+1 WHERE user_id=$1", id)
	return err
}
func (s Store) Confirm(ctx context.Context, tx *sqlx.Tx, id int, v Pending) error {
	r, err := tx.ExecContext(ctx, "UPDATE st_users SET email=$2 WHERE id=$1 AND lower(email)=lower($3)", id, v.NewEmail, v.OldEmail)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM st_email_changes WHERE user_id=$1", id); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM st_token WHERE user_id=$1", id)
	return err
}
func (s Store) Discard(ctx context.Context, id int, newDigest []byte) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM st_email_changes WHERE user_id=$1 AND new_digest=$2", id, newDigest)
	return err
}
