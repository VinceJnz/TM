package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	ResourceQryGetAll = `SELECT * FROM et_resource`
	ResourceQryGet    = `SELECT * FROM et_resource WHERE id = $1`
	ResourceQryCreate = `INSERT INTO et_resource (name, description)
					VALUES ($1, $2)
					RETURNING id`
	ResourceQryUpdate = `UPDATE et_resource 
					SET (name, description) = ($1, $2)
					WHERE id = $3`
	ResourceQryDelete = `DELETE FROM et_resource WHERE id = $1`
)

func NewResource(db *sqlx.DB) Store[models.Resource] {
	return Store[models.Resource]{DB: db, Definition: Definition[models.Resource]{Queries: Queries{List: explicit[models.Resource](ResourceQryGetAll), Get: explicit[models.Resource](ResourceQryGet), Create: ResourceQryCreate, Update: ResourceQryUpdate, Delete: ResourceQryDelete}, CreateArgs: func(record *models.Resource) []any { return []any{record.Name, record.Description} }, UpdateArgs: func(id int, record *models.Resource) []any { return []any{record.Name, record.Description, id} }, Limits: []int{45, 45}}}
}
