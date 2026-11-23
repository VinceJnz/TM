package authentication

import (
	"api-server/v2/models"
	"context"
	"fmt"
)

const capabilityQuery = `SELECT etat.ID, etat.name AS access_scope, stg.name AS "group"
	FROM st_users stu
		JOIN st_user_group stug ON stug.user_id=stu.id
		JOIN st_group stg ON stg.id=stug.group_id
		JOIN st_group_resource stgr ON stgr.group_id=stg.id
		JOIN et_resource etr ON etr.id=stgr.resource_id
		JOIN et_access_level etal ON etal.id=stgr.access_level_id
		JOIN et_access_scope etat ON etat.id=stgr.access_scope_id
	WHERE stu.ID=$1
		AND stu.user_account_status_id=$4
		AND UPPER(etr.Name)=UPPER($2)
		AND UPPER(etal.Name)=UPPER($3)
	GROUP BY etat.ID, etat.name, stg.name
	ORDER BY
		CASE LOWER(etat.name::text)
			WHEN 'any' THEN 2
			WHEN 'own' THEN 1
			ELSE 0
		END DESC,
		etat.ID DESC
	LIMIT 1`

// Capability returns the strongest matching grant for an active account.
func (s Store) Capability(ctx context.Context, userID int, resource, method string) (models.AccessCheck, error) {
	var grant models.AccessCheck
	err := s.DB.QueryRowContext(ctx, capabilityQuery, userID, resource, method, models.AccountActive).
		Scan(&grant.AccessScopeID, &grant.AccessScope, &grant.Group)
	if err != nil {
		return models.AccessCheck{}, fmt.Errorf("access lookup: %w", err)
	}
	return grant, nil
}
