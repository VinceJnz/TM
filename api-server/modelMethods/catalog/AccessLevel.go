package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	AccessLevelQryGetAll = `SELECT * FROM et_access_level`
	AccessLevelQryGet    = `SELECT * FROM et_access_level WHERE id = $1`
	AccessLevelQryCreate = `INSERT INTO et_access_level (name, description)
					VALUES ($1, $2)
					RETURNING id`
	AccessLevelQryUpdate = `UPDATE et_access_level 
					SET name = $1, description = $2
					WHERE id = $3`
	AccessLevelQryDelete = `DELETE FROM et_access_level WHERE id = $1`
)

func NewAccessLevel(db *sqlx.DB) Store[models.AccessLevel] {
	return Store[models.AccessLevel]{DB: db, Definition: Definition[models.AccessLevel]{Queries: Queries{List: explicit[models.AccessLevel](AccessLevelQryGetAll), Get: explicit[models.AccessLevel](AccessLevelQryGet), Create: AccessLevelQryCreate, Update: AccessLevelQryUpdate, Delete: AccessLevelQryDelete}, CreateArgs: func(record *models.AccessLevel) []any { return []any{record.Name, record.Description} }, UpdateArgs: func(id int, record *models.AccessLevel) []any { return []any{record.Name, record.Description, id} }, Limits: []int{45, 45}}}
}
