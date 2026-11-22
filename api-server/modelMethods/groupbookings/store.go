package groupbookings

import (
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }

const read = `SELECT id,group_name,owner_id,created,modified FROM at_group_bookings`

func (s Store) List(ctx context.Context, p resourcequery.Page) ([]models.GroupBooking, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[models.GroupBooking](ctx, s.DB, p, read)
}
func (s Store) Get(ctx context.Context, id int) (models.GroupBooking, error) {
	var v models.GroupBooking
	err := s.DB.GetContext(ctx, &v, read+" WHERE id=$1", id)
	return v, err
}
func (s Store) Lock(ctx context.Context, tx *sqlx.Tx, id int) (models.GroupBooking, error) {
	var v models.GroupBooking
	err := tx.GetContext(ctx, &v, read+" WHERE id=$1 FOR UPDATE", id)
	return v, err
}
func (s Store) Referenced(ctx context.Context, tx *sqlx.Tx, id int) (bool, error) {
	var yes bool
	err := tx.GetContext(ctx, &yes, "SELECT EXISTS(SELECT 1 FROM at_bookings WHERE group_booking_id=$1)", id)
	return yes, err
}
func (s Store) Save(ctx context.Context, tx *sqlx.Tx, id int, v models.GroupBookingWrite) (models.GroupBooking, error) {
	var result models.GroupBooking
	q := `INSERT INTO at_group_bookings(group_name,owner_id) VALUES($1,$2) RETURNING id,group_name,owner_id,created,modified`
	args := []any{v.GroupName, v.OwnerID}
	if id > 0 {
		q = `UPDATE at_group_bookings SET group_name=$1,owner_id=$2 WHERE id=$3 RETURNING id,group_name,owner_id,created,modified`
		args = append(args, id)
	}
	err := tx.GetContext(ctx, &result, q, args...)
	return result, err
}
func (s Store) Delete(ctx context.Context, tx *sqlx.Tx, id int) error {
	r, err := tx.ExecContext(ctx, "DELETE FROM at_group_bookings WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
