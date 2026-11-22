package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	AccessScopeQryGetAll = `SELECT * FROM et_access_scope`
	AccessScopeQryGet    = `SELECT * FROM et_access_scope WHERE id = $1`
	AccessScopeQryCreate = `INSERT INTO et_access_scope (name, description)
					VALUES ($1, $2)
					RETURNING id`
	AccessScopeQryUpdate = `UPDATE et_access_scope 
					SET name = $1, description = $2
					WHERE id = $3`
	AccessScopeQryDelete = `DELETE FROM et_access_scope WHERE id = $1`
)

func NewAccessScope(db *sqlx.DB) Store[models.AccessScope] {
	return Store[models.AccessScope]{DB: db, Definition: Definition[models.AccessScope]{Queries: Queries{List: explicit[models.AccessScope](AccessScopeQryGetAll), Get: explicit[models.AccessScope](AccessScopeQryGet), Create: AccessScopeQryCreate, Update: AccessScopeQryUpdate, Delete: AccessScopeQryDelete}, CreateArgs: func(record *models.AccessScope) []any { return []any{record.Name, record.Description} }, UpdateArgs: func(id int, record *models.AccessScope) []any { return []any{record.Name, record.Description, id} }, Limits: []int{45, 45}}}
}
