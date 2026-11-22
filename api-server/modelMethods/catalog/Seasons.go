package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	SeasonsQryGetAll = `SELECT * FROM et_seasons`
	SeasonsQryGet    = `SELECT * FROM et_seasons WHERE id = $1`
	SeasonsQryCreate = `INSERT INTO et_seasons (season)
					VALUES ($1)
					RETURNING id`
	SeasonsQryUpdate = `UPDATE et_seasons 
					SET season = $1 
					WHERE id = $2`
	SeasonsQryDelete = `DELETE FROM et_seasons WHERE id = $1`
)

func NewSeasons(db *sqlx.DB) Store[models.Season] {
	return Store[models.Season]{DB: db, Definition: Definition[models.Season]{Queries: Queries{List: explicit[models.Season](SeasonsQryGetAll), Get: explicit[models.Season](SeasonsQryGet), Create: SeasonsQryCreate, Update: SeasonsQryUpdate, Delete: SeasonsQryDelete}, CreateArgs: func(record *models.Season) []any { return []any{record.Season} }, UpdateArgs: func(id int, record *models.Season) []any { return []any{record.Season, id} }, Limits: []int{255}}}
}
