package catalog

import (
	"api-server/v2/models"
	"github.com/jmoiron/sqlx"
)

const (
	UserAgeGroupsQryGetAll = `SELECT * FROM et_user_age_groups`
	UserAgeGroupsQryGet    = `SELECT * FROM et_user_age_groups WHERE id = $1`
	UserAgeGroupsQryCreate = `INSERT INTO et_user_age_groups (age_group)
					VALUES ($1)
					RETURNING id`
	UserAgeGroupsQryUpdate = `UPDATE et_user_age_groups 
					SET age_group = $1
					WHERE id = $3`
	UserAgeGroupsQryDelete = `DELETE FROM et_user_age_groups WHERE id = $1`
)

func NewUserAgeGroups(db *sqlx.DB) Store[models.UserAgeGroups] {
	return Store[models.UserAgeGroups]{DB: db, Definition: Definition[models.UserAgeGroups]{Queries: Queries{List: explicit[models.UserAgeGroups](UserAgeGroupsQryGetAll), Get: explicit[models.UserAgeGroups](UserAgeGroupsQryGet), Create: UserAgeGroupsQryCreate, Update: UserAgeGroupsQryUpdate, Delete: UserAgeGroupsQryDelete}, CreateArgs: func(record *models.UserAgeGroups) []any { return []any{record.AgeGroup} }, UpdateArgs: func(id int, record *models.UserAgeGroups) []any { return []any{record.AgeGroup, id} }, Limits: []int{255}}}
}
