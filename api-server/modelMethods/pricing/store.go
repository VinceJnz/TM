// Package pricing owns trip-price SQL and deliberately has no HTTP dependencies.
package pricing

import (
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }

const costColumns = "id,trip_cost_group_id,description,member_status_id,user_age_group_id,season_id,amount,created,modified"
const groupColumns = "id,description,created,modified"

func (s Store) Costs(ctx context.Context, p resourcequery.Page) ([]models.TripCost, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[models.TripCost](ctx, s.DB, p, `SELECT c.id,c.trip_cost_group_id,c.description,c.member_status_id,m.status AS member_status,c.user_age_group_id,a.age_group AS user_age_group,c.season_id,s.season,c.amount,c.created,c.modified FROM at_trip_costs c LEFT JOIN et_member_status m ON m.id=c.member_status_id JOIN et_user_age_groups a ON a.id=c.user_age_group_id JOIN et_seasons s ON s.id=c.season_id`)
}
func (s Store) Groups(ctx context.Context, p resourcequery.Page) ([]models.TripCostGroup, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[models.TripCostGroup](ctx, s.DB, p, "SELECT "+groupColumns+" FROM at_trip_cost_groups")
}
func (s Store) Cost(ctx context.Context, id int) (models.TripCost, error) {
	var v models.TripCost
	err := s.DB.GetContext(ctx, &v, "SELECT "+costColumns+" FROM at_trip_costs WHERE id=$1", id)
	return v, err
}
func (s Store) Group(ctx context.Context, id int) (models.TripCostGroup, error) {
	var v models.TripCostGroup
	err := s.DB.GetContext(ctx, &v, "SELECT "+groupColumns+" FROM at_trip_cost_groups WHERE id=$1", id)
	return v, err
}
func (s Store) SaveCost(ctx context.Context, id int, v models.TripCostWrite) (models.TripCost, error) {
	var result models.TripCost
	query := "INSERT INTO at_trip_costs(trip_cost_group_id,member_status_id,user_age_group_id,season_id,amount) VALUES($1,$2,$3,$4,$5) RETURNING " + costColumns
	args := []any{v.TripCostGroupID, v.MemberStatusID, v.UserAgeGroupID, v.SeasonID, v.Amount}
	if id > 0 {
		query = "UPDATE at_trip_costs SET trip_cost_group_id=$1,member_status_id=$2,user_age_group_id=$3,season_id=$4,amount=$5,modified=CURRENT_TIMESTAMP WHERE id=$6 RETURNING " + costColumns
		args = append(args, id)
	}
	err := s.DB.GetContext(ctx, &result, query, args...)
	return result, err
}
func (s Store) SaveGroup(ctx context.Context, id int, v models.TripCostGroupWrite) (models.TripCostGroup, error) {
	var result models.TripCostGroup
	query := "INSERT INTO at_trip_cost_groups(description) VALUES($1) RETURNING " + groupColumns
	args := []any{v.Description}
	if id > 0 {
		query = "UPDATE at_trip_cost_groups SET description=$1,modified=CURRENT_TIMESTAMP WHERE id=$2 RETURNING " + groupColumns
		args = append(args, id)
	}
	err := s.DB.GetContext(ctx, &result, query, args...)
	return result, err
}
func (s Store) DeleteCost(ctx context.Context, id int) error {
	return s.remove(ctx, "DELETE FROM at_trip_costs WHERE id=$1", id)
}
func (s Store) DeleteGroup(ctx context.Context, id int) error {
	return s.remove(ctx, "DELETE FROM at_trip_cost_groups WHERE id=$1", id)
}
func (s Store) remove(ctx context.Context, query string, id int) error {
	r, e := s.DB.ExecContext(ctx, query, id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e == nil && n == 0 {
		return sql.ErrNoRows
	}
	return e
}
