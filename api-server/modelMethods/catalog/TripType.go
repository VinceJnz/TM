package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	TripTypeQryGetAll = `SELECT * FROM et_trip_type`
	TripTypeQryGet    = `SELECT * FROM et_trip_type WHERE id = $1`
	TripTypeQryCreate = `INSERT INTO et_trip_type (type)
					VALUES ($1)
					RETURNING id`
	TripTypeQryUpdate = `UPDATE et_trip_type 
					SET type = $1 
					WHERE id = $2`
	TripTypeQryDelete = `DELETE FROM et_trip_type WHERE id = $1`
)

func NewTripType(db *sqlx.DB) Store[models.TripType] {
	return Store[models.TripType]{DB: db, Definition: Definition[models.TripType]{Queries: Queries{List: explicit[models.TripType](TripTypeQryGetAll), Get: explicit[models.TripType](TripTypeQryGet), Create: TripTypeQryCreate, Update: TripTypeQryUpdate, Delete: TripTypeQryDelete}, CreateArgs: func(record *models.TripType) []any { return []any{record.Type} }, UpdateArgs: func(id int, record *models.TripType) []any { return []any{record.Type, id} }, Limits: []int{255}}}
}
