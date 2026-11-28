package security

import (
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
)

const groupRead = `SELECT id,version,COALESCE(name,'') AS name,COALESCE(description,'') AS description FROM st_group`

var ErrReferencedGroup = errors.New("group still has memberships or grants")
var ErrBuiltInGroup = errors.New("built-in group cannot be deleted")

func (s Store) Groups(ctx context.Context, p resourcequery.Page) ([]models.Group, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[models.Group](ctx, s.DB, p, groupRead)
}
func (s Store) Group(ctx context.Context, id int) (models.Group, error) {
	var v models.Group
	err := s.DB.GetContext(ctx, &v, groupRead+" WHERE id=$1", id)
	return v, err
}
func (s Store) SaveGroup(ctx context.Context, tx *sqlx.Tx, id int, v models.GroupWrite) (models.Group, error) {
	var result models.Group
	query := `INSERT INTO st_group(name,description) VALUES($1,$2) RETURNING id,version,name,description`
	args := []any{v.Name, v.Description}
	if id > 0 {
		query = `UPDATE st_group SET name=$1,description=$2 WHERE id=$3 AND version=$4 RETURNING id,version,name,description`
		args = append(args, id, v.Version)
	}
	err := tx.GetContext(ctx, &result, query, args...)
	if id > 0 && errors.Is(err, sql.ErrNoRows) {
		return result, models.ErrStaleRecord
	}
	return result, err
}
func (s Store) DeleteGroup(ctx context.Context, tx *sqlx.Tx, id int) error {
	// FOR UPDATE conflicts with the key-share lock taken by FK inserts. It
	// closes the gap between checking dependencies and the legacy CASCADE delete.
	var builtIn bool
	if err := tx.GetContext(ctx, &builtIn, "SELECT system_key IS NOT NULL FROM st_group WHERE id=$1 FOR UPDATE", id); err != nil {
		return err
	}
	if builtIn {
		return ErrBuiltInGroup
	}
	var referenced bool
	if err := tx.GetContext(ctx, &referenced, `SELECT EXISTS(SELECT 1 FROM st_user_group WHERE group_id=$1) OR EXISTS(SELECT 1 FROM st_group_resource WHERE group_id=$1)`, id); err != nil {
		return err
	}
	if referenced {
		return ErrReferencedGroup
	}
	return remove(ctx, tx, "DELETE FROM st_group WHERE id=$1", id)
}
