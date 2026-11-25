package trips

import (
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"time"
)

type Store struct{ DB *sqlx.DB }

type BookingChoice struct {
	ID       int       `json:"id" db:"id"`
	Name     string    `json:"trip_name" db:"trip_name"`
	FromDate time.Time `json:"from_date" db:"from_date"`
	ToDate   time.Time `json:"to_date" db:"to_date"`
}

func (s Store) BookingChoices(ctx context.Context, page resourcequery.Page) ([]BookingChoice, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[BookingChoice](ctx, s.DB, page, `SELECT id,trip_name,from_date,to_date FROM at_trips WHERE from_date IS NOT NULL AND to_date IS NOT NULL`)
}

func (s Store) List(ctx context.Context, page resourcequery.Page) ([]models.Trip, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[models.Trip](ctx, s.DB, page, qryGetAll)
}
func (s Store) Get(ctx context.Context, id int) (models.Trip, error) {
	var trip models.Trip
	err := s.DB.GetContext(ctx, &trip, qryGet, id)
	return trip, err
}
func (s Store) Lock(ctx context.Context, tx *sqlx.Tx, id, actor int, any bool) error {
	var locked int
	return tx.GetContext(ctx, &locked, `SELECT id FROM at_trips WHERE id=$1 AND (owner_id=$2 OR $3) FOR UPDATE`, id, actor, any)
}
func (s Store) BookingsFit(ctx context.Context, tx *sqlx.Tx, input models.Trip) (bool, error) {
	var valid bool
	err := tx.GetContext(ctx, &valid, `SELECT NOT EXISTS(SELECT 1 FROM at_bookings WHERE trip_id=$1 AND booking_status_id<>2 AND (from_date<$2 OR to_date>$3))`, input.ID, input.FromDate, input.ToDate)
	return valid, err
}
func (s Store) Create(ctx context.Context, tx *sqlx.Tx, actor int, input models.Trip) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, qryCreate, actor, input.Name, input.Location, input.DifficultyID, input.FromDate, input.ToDate, input.MaxParticipants, input.TripStatusID, input.TripTypeID, input.TripCostGroupID, descriptionValue(input)).Scan(&id)
	return id, err
}
func changed(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}
func descriptionValue(input models.Trip) interface{} {
	if !input.Description.Valid {
		return nil
	}
	return input.Description.String
}

func (s Store) Update(ctx context.Context, tx *sqlx.Tx, actor int, any bool, input *models.Trip) error {
	scope := "own"
	if any {
		scope = "any"
	}
	return tx.QueryRowContext(ctx, qryUpdate, input.ID, actor, scope, input.Name, input.Location, input.DifficultyID, input.FromDate, input.ToDate, input.MaxParticipants, input.TripStatusID, input.TripTypeID, input.TripCostGroupID, descriptionValue(*input)).Scan(&input.Description)
}
func (s Store) Delete(ctx context.Context, tx *sqlx.Tx, actor int, any bool, id int) error {
	scope := "own"
	if any {
		scope = "any"
	}
	return changed(tx.ExecContext(ctx, qryDelete, id, actor, scope))
}

func (s Store) Version(ctx context.Context, tx *sqlx.Tx, id int) (int64, error) {
	var version int64
	err := tx.GetContext(ctx, &version, `SELECT version FROM at_trips WHERE id=$1`, id)
	return version, err
}
