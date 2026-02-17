package handlerAuth

import (
	"encoding/json"

	"api-server/v2/modelMethods/dbAuthTemplate"
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

// MenuUserGet is a public endpoint that returns menu user data if authenticated, or empty data if not
// This allows the client to check authentication status on page load without getting errors
func (h *Handler) MenuUserGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to find session cookie
	sc, err := r.Cookie("session")
	if err != nil {
		// No session cookie - return empty response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.MenuUser{})
		return
	}

	// Check if session is valid
	dbToken, err := dbAuthTemplate.FindSessionToken(debugTag+"MenuUserGet", h.appConf.Db, sc.Value)
	if err != nil {
		// Session cookie exists but is invalid/expired
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.MenuUser{})
		return
	}

	// Load user info
	user, err := dbAuthTemplate.UserReadQry(debugTag+"MenuUserGet:user", h.appConf.Db, dbToken.UserID)
	if err != nil || !user.UserActive() {
		// User not found or not active
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.MenuUser{})
		return
	}

	// User is authenticated and active - return menuUser info
	var menuUser models.MenuUser
	err = h.appConf.Db.Get(&menuUser, sqlMenuUser, dbToken.UserID, models.AccountActive)
	if err != nil {
		// Failed to get menu user (maybe no group assigned)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.MenuUser{})
		return
	}

	// Return menu user info
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(menuUser)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) MenuListGet(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.MenuItem{}, sqlMenuList, session.UserID, models.AccountActive)
}

// AuthStatus checks if the user is authenticated and returns their info, or returns not_authenticated status
// This endpoint does not require authentication and won't return 401
func (h *Handler) AuthStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to find session cookie
	sc, err := r.Cookie("session")
	if err != nil {
		// No session cookie - user not authenticated
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": false,
			"user":          nil,
		})
		return
	}

	// Check if session is valid
	dbToken, err := dbAuthTemplate.FindSessionToken(debugTag+"AuthStatus", h.appConf.Db, sc.Value)
	if err != nil {
		// Session cookie exists but is invalid/expired
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": false,
			"user":          nil,
		})
		return
	}

	// Load user info
	user, err := dbAuthTemplate.UserReadQry(debugTag+"AuthStatus:user", h.appConf.Db, dbToken.UserID)
	if err != nil || !user.UserActive() {
		// User not found or not active
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": false,
			"user":          nil,
		})
		return
	}

	// User is authenticated - return menuUser info
	var menuUser models.MenuUser
	err = h.appConf.Db.Get(&menuUser, sqlMenuUser, dbToken.UserID, models.AccountActive)
	if err != nil {
		// Failed to get menu user (maybe no group assigned)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": true,
			"user": map[string]any{
				"user_id":    user.ID,
				"name":       user.Name,
				"email":      user.Email.String,
				"group":      "",
				"admin_flag": false,
			},
		})
		return
	}

	// Return full menu user info
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true,
		"user":          menuUser,
	})
}
