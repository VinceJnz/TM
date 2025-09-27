package handlerAuth

import (
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"net/http"
)

const (
	sqlMenuUser = `SELECT stu.ID AS user_id, stu.name, stg.name AS group, stg.admin_flag
		FROM st_users stu
			JOIN st_user_group stug ON stug.User_ID=stu.ID
			JOIN st_group stg ON stg.ID=stug.Group_ID
		WHERE stu.ID=$1
			AND stu.user_account_status_id=$2
		ORDER BY stg.admin_flag -- This might need to change to DESC
		LIMIT 1`

	sqlMenuList = `SELECT stu.ID AS user_id, etr.Name AS resource, stgr.admin_flag
		FROM st_users stu
			JOIN st_user_group stug ON stug.User_ID=stu.ID
			JOIN st_group stg ON stg.ID=stug.Group_ID
			JOIN st_group_resource stgr ON stgr.Group_ID=stg.ID
			JOIN et_resource etr ON etr.ID=stgr.Resource_ID
		WHERE stu.ID=$1
			AND stu.user_account_status_id=$2
		--ORDER BY stgr.admin_flag -- This might need to change to DESC
		GROUP BY stu.ID, etr.Name, stgr.admin_flag`
)

// Get: retrieves and returns a single record identified by id
func (h *Handler) MenuUserGet(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	//log.Printf(debugTag+"MenuUserGet()1 session=%+v", session)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &models.MenuUser{}, sqlMenuUser, session.UserID, models.AccountActive)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) MenuListGet(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	//log.Printf(debugTag+"MenuListGet()1 session=%+v", session)
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.MenuItem{}, sqlMenuList, session.UserID, models.AccountActive)
}
