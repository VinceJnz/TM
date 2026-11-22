package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	BookingStatusQryGetAll = `SELECT * FROM et_booking_status`
	BookingStatusQryGet    = `SELECT * FROM et_booking_status WHERE id = $1`
	BookingStatusQryCreate = `INSERT INTO et_booking_status (status)
					VALUES ($1)
					RETURNING id`
	BookingStatusQryUpdate = `UPDATE et_booking_status 
					SET status = $1 
					WHERE id = $2`
	BookingStatusQryDelete = `DELETE FROM et_booking_status WHERE id = $1`
)

func NewBookingStatus(db *sqlx.DB) Store[models.BookingStatus] {
	return Store[models.BookingStatus]{DB: db, Definition: Definition[models.BookingStatus]{Queries: Queries{List: explicit[models.BookingStatus](BookingStatusQryGetAll), Get: explicit[models.BookingStatus](BookingStatusQryGet), Create: BookingStatusQryCreate, Update: BookingStatusQryUpdate, Delete: BookingStatusQryDelete}, CreateArgs: func(record *models.BookingStatus) []any { return []any{record.Status} }, UpdateArgs: func(id int, record *models.BookingStatus) []any { return []any{record.Status, id} }, Limits: []int{50}}}
}
