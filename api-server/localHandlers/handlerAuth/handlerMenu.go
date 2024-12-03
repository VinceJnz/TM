package handlerAuth

import (
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"
	"net/http"
)

const (
	sqlUserMenuAccess = `SELECT stu.ID, stg.Name AS group, stg.admin_flag
		FROM st_users stu
			JOIN st_user_group stug ON stug.User_ID=stu.ID
			JOIN st_group stg ON stg.ID=stug.Group_ID
		WHERE stu.ID=$1
			AND stu.User_status_ID=1
		ORDER BY stg.admin_flag -- This might need to change to DESC
		LIMIT 1`

	sqlUserMenuAccessList = `SELECT stu.ID, stg.Name AS group, stg.admin_flag, etr.Name AS resource
		FROM st_users stu
			JOIN st_user_group stug ON stug.User_ID=stu.ID
			JOIN st_group stg ON stg.ID=stug.Group_ID
			JOIN st_group_resource stgr ON stgr.Group_ID=stg.ID
			JOIN et_resource etr ON etr.ID=stgr.Resource_ID
		WHERE stu.ID=$1
			AND stu.User_status_ID=1`
)

// Get: retrieves and returns a single record identified by id
func (h *Handler) MenuGet(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.User{}, sqlUserMenuAccess, id)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) MenuListGet(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.User{}, sqlUserMenuAccessList, id)
}
