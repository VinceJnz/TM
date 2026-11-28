package authentication

import (
	"api-server/v2/models"
	"context"
)

const (
	sqlMenuUser = `SELECT stu.ID AS user_id, stu.name, COALESCE(stg.name::text, '') AS "group"
		FROM st_users stu
			LEFT JOIN st_user_group stug ON stug.User_ID=stu.ID
			LEFT JOIN st_group stg ON stg.ID=stug.Group_ID
		WHERE stu.ID=$1
			AND stu.user_account_status_id=$2
		ORDER BY
			CASE stg.system_key
				WHEN 'sysadmin' THEN 4
				WHEN 'admin' THEN 3
				WHEN 'trustedusers' THEN 2
				WHEN 'user' THEN 1
				ELSE 0
			END DESC,
			stg.ID DESC
		LIMIT 1`

	sqlMenuList = `SELECT stu.ID AS user_id, etr.Name AS resource, etal.name AS access_level, etas.name AS access_scope
		FROM st_users stu
			JOIN st_user_group stug ON stug.User_ID=stu.ID
			JOIN st_group stg ON stg.ID=stug.Group_ID
			JOIN st_group_resource stgr ON stgr.Group_ID=stg.ID
			JOIN et_resource etr ON etr.ID=stgr.Resource_ID
			JOIN et_access_level etal ON etal.id=stgr.access_level_id
			JOIN et_access_scope etas ON etas.id=stgr.access_scope_id
		WHERE stu.ID=$1
			AND stu.user_account_status_id=$2
		GROUP BY stu.ID, etr.Name, etal.name, etas.name`
)

func (s Store) MenuUser(ctx context.Context, userID int) (models.MenuUser, error) {
	var item models.MenuUser
	err := s.DB.GetContext(ctx, &item, sqlMenuUser, userID, models.AccountActive)
	return item, err
}

func (s Store) MenuCapabilities(ctx context.Context, userID int) ([]models.MenuItem, error) {
	items := []models.MenuItem{}
	err := s.DB.SelectContext(ctx, &items, sqlMenuList, userID, models.AccountActive)
	return items, err
}
