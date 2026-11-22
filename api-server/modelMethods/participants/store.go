package participants

import (
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }

func (s Store) List(ctx context.Context, actor int, scope string, parent, id int, p resourcequery.Page) ([]models.BookingPeople, resourcequery.PageInfo, error) {
	if id > 0 {
		return resourcequery.SelectPage[models.BookingPeople](ctx, s.DB, p, qryGet, id, actor, scope)
	}
	if parent > 0 {
		return resourcequery.SelectPage[models.BookingPeople](ctx, s.DB, p, qryGetList, parent, actor, scope)
	}
	return resourcequery.SelectPage[models.BookingPeople](ctx, s.DB, p, qryGetAll, actor, scope)
}
func (s Store) Get(ctx context.Context, actor int, scope string, id int) (models.BookingPeople, error) {
	var v models.BookingPeople
	err := s.DB.GetContext(ctx, &v, qryGet, id, actor, scope)
	return v, err
}
func (s Store) Parent(ctx context.Context, tx *sqlx.Tx, id int) (models.Booking, error) {
	var v models.Booking
	var trip, locked int
	if err := tx.GetContext(ctx, &trip, "SELECT trip_id FROM at_bookings WHERE id=$1", id); err != nil {
		return v, err
	}
	if err := tx.GetContext(ctx, &locked, "SELECT id FROM at_trips WHERE id=$1 FOR UPDATE", trip); err != nil {
		return v, err
	}
	err := tx.GetContext(ctx, &v, `SELECT id,trip_id,owner_id,booking_status_id,stripe_session_id FROM at_bookings WHERE id=$1 FOR UPDATE`, id)
	if err == nil && v.TripID.Int64 != int64(trip) {
		return v, models.ErrStaleRecord
	}
	return v, err
}
func (s Store) Save(ctx context.Context, tx *sqlx.Tx, actor int, scope string, id int, v models.ParticipantWrite) (int, error) {
	if id == 0 {
		err := tx.GetContext(ctx, &id, qryCreate, actor, v.BookingID, v.PersonID, v.Notes, scope)
		return id, err
	}
	r, err := tx.ExecContext(ctx, qryUpdate, id, actor, scope, v.PersonID, v.Notes)
	return id, changed(r, err)
}
func (s Store) Delete(ctx context.Context, tx *sqlx.Tx, actor int, scope string, id int) error {
	r, err := tx.ExecContext(ctx, qryDelete, id, actor, scope)
	return changed(r, err)
}
func changed(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
