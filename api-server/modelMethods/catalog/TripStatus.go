package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	TripStatusQryGetAll = `SELECT * FROM et_trip_status`
	TripStatusQryGet    = `SELECT * FROM et_trip_status WHERE id = $1`
	TripStatusQryCreate = `INSERT INTO et_trip_status (status)
					VALUES ($1)
					RETURNING id`
	TripStatusQryUpdate = `UPDATE et_trip_status 
					SET status = $1 
					WHERE id = $2`
	TripStatusQryDelete = `DELETE FROM et_trip_status WHERE id = $1`
)

func NewTripStatus(db *sqlx.DB) Store[models.TripStatus] {
	return Store[models.TripStatus]{DB: db, Definition: Definition[models.TripStatus]{Queries: Queries{List: explicit[models.TripStatus](TripStatusQryGetAll), Get: explicit[models.TripStatus](TripStatusQryGet), Create: TripStatusQryCreate, Update: TripStatusQryUpdate, Delete: TripStatusQryDelete}, CreateArgs: func(record *models.TripStatus) []any { return []any{record.Status} }, UpdateArgs: func(id int, record *models.TripStatus) []any { return []any{record.Status, id} }, Limits: []int{50}}}
}
