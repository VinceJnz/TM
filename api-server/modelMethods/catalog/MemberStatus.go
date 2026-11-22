package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	MemberStatusQryGetAll = `SELECT * FROM et_member_status`
	MemberStatusQryGet    = `SELECT * FROM et_member_status WHERE id = $1`
	MemberStatusQryCreate = `INSERT INTO et_member_status (status)
					VALUES ($1)
					RETURNING id`
	MemberStatusQryUpdate = `UPDATE et_member_status 
					SET status = $1 
					WHERE id = $2`
	MemberStatusQryDelete = `DELETE FROM et_member_status WHERE id = $1`
)

func NewMemberStatus(db *sqlx.DB) Store[models.MemberStatus] {
	return Store[models.MemberStatus]{DB: db, Definition: Definition[models.MemberStatus]{Queries: Queries{List: explicit[models.MemberStatus](MemberStatusQryGetAll), Get: explicit[models.MemberStatus](MemberStatusQryGet), Create: MemberStatusQryCreate, Update: MemberStatusQryUpdate, Delete: MemberStatusQryDelete}, CreateArgs: func(record *models.MemberStatus) []any { return []any{record.Status} }, UpdateArgs: func(id int, record *models.MemberStatus) []any { return []any{record.Status, id} }, Limits: []int{255}}}
}
