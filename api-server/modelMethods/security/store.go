// Package security owns capability and membership persistence.
package security

import (
	"api-server/v2/internal/observability"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"strconv"
)

type Store struct{ DB *sqlx.DB }

const grantRead = `SELECT r.id,r.version,r.group_id,g.name AS group_name,r.resource_id,e.name AS resource,r.access_level_id,l.name AS access_level,r.access_scope_id,s.name AS access_scope,r.created,r.modified FROM st_group_resource r JOIN st_group g ON g.id=r.group_id JOIN et_resource e ON e.id=r.resource_id JOIN et_access_level l ON l.id=r.access_level_id JOIN et_access_scope s ON s.id=r.access_scope_id`
const memberRead = `SELECT m.id,m.version,m.user_id,u.name AS user_name,m.group_id,g.name AS group_name,m.created,m.modified FROM st_user_group m JOIN st_users u ON u.id=m.user_id JOIN st_group g ON g.id=m.group_id`

func (s Store) Grants(ctx context.Context, p resourcequery.Page, group ...int) ([]models.GroupResource, resourcequery.PageInfo, error) {
	if len(group) > 0 && group[0] > 0 {
		return resourcequery.SelectPage[models.GroupResource](ctx, s.DB, p, grantRead+" WHERE r.group_id=$1", group[0])
	}
	return resourcequery.SelectPage[models.GroupResource](ctx, s.DB, p, grantRead)
}
func (s Store) Members(ctx context.Context, p resourcequery.Page, user ...int) ([]models.UserGroup, resourcequery.PageInfo, error) {
	if len(user) > 0 && user[0] > 0 {
		return resourcequery.SelectPage[models.UserGroup](ctx, s.DB, p, memberRead+" WHERE m.user_id=$1", user[0])
	}
	return resourcequery.SelectPage[models.UserGroup](ctx, s.DB, p, memberRead)
}
func (s Store) Grant(ctx context.Context, id int) (models.GroupResource, error) {
	var v models.GroupResource
	err := s.DB.GetContext(ctx, &v, grantRead+" WHERE r.id=$1", id)
	return v, err
}
func (s Store) Member(ctx context.Context, id int) (models.UserGroup, error) {
	var v models.UserGroup
	err := s.DB.GetContext(ctx, &v, memberRead+" WHERE m.id=$1", id)
	return v, err
}
func (s Store) Begin(ctx context.Context, actor int) (*sqlx.Tx, error) {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, "SELECT set_config('tm.actor',$1,true),set_config('tm.request_id',$2,true)", "user:"+strconv.Itoa(actor), observability.RequestID(ctx))
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}
func (s Store) GrantShape(ctx context.Context, tx *sqlx.Tx, v models.GroupResourceWrite) (resource, method, scope string, err error) {
	err = tx.QueryRowxContext(ctx, `SELECT r.name,l.name,s.name FROM et_resource r CROSS JOIN et_access_level l CROSS JOIN et_access_scope s WHERE r.id=$1 AND l.id=$2 AND s.id=$3`, v.ResourceID, v.AccessLevelID, v.AccessScopeID).Scan(&resource, &method, &scope)
	return
}
func (s Store) SaveGrant(ctx context.Context, tx *sqlx.Tx, id int, v models.GroupResourceWrite) (models.GroupResource, error) {
	var result models.GroupResource
	query := "INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id) VALUES($1,$2,$3,$4) RETURNING id,version,group_id,resource_id,access_level_id,access_scope_id,created,modified"
	args := []any{v.GroupID, v.ResourceID, v.AccessLevelID, v.AccessScopeID}
	if id > 0 {
		query = "UPDATE st_group_resource SET group_id=$1,resource_id=$2,access_level_id=$3,access_scope_id=$4,modified=CURRENT_TIMESTAMP WHERE id=$5 AND version=$6 RETURNING id,version,group_id,resource_id,access_level_id,access_scope_id,created,modified"
		args = append(args, id, v.Version)
	}
	err := tx.GetContext(ctx, &result, query, args...)
	if id > 0 && err == sql.ErrNoRows {
		return result, models.ErrStaleRecord
	}
	return result, err
}
func (s Store) SaveMember(ctx context.Context, tx *sqlx.Tx, id int, v models.UserGroupWrite) (models.UserGroup, error) {
	var result models.UserGroup
	query := "INSERT INTO st_user_group(user_id,group_id) VALUES($1,$2) RETURNING id,version,user_id,group_id,created,modified"
	args := []any{v.UserID, v.GroupID}
	if id > 0 {
		query = "UPDATE st_user_group SET user_id=$1,group_id=$2,modified=CURRENT_TIMESTAMP WHERE id=$3 AND version=$4 RETURNING id,version,user_id,group_id,created,modified"
		args = append(args, id, v.Version)
	}
	err := tx.GetContext(ctx, &result, query, args...)
	if id > 0 && err == sql.ErrNoRows {
		return result, models.ErrStaleRecord
	}
	return result, err
}
func (s Store) DeleteGrant(ctx context.Context, tx *sqlx.Tx, id int) error {
	return remove(ctx, tx, "DELETE FROM st_group_resource WHERE id=$1", id)
}
func (s Store) DeleteMember(ctx context.Context, tx *sqlx.Tx, id int) error {
	return remove(ctx, tx, "DELETE FROM st_user_group WHERE id=$1", id)
}
func remove(ctx context.Context, tx *sqlx.Tx, query string, id int) error {
	r, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
