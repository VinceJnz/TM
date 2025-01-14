package handlerAuth

import (
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/models"
	"errors"
	"log"
)

// sqlCheckAccess checks that the user account has been activated and that it has access to the requested resource and method.
// NOTE: If the group (stg) admin_flag is set then access is given regardless of the resource or action settings.
const (
	sqlUserCheckAccess = `SELECT etat.ID, (stgr.admin_flag OR stg.admin_flag) AS admin_flag
	FROM st_users stu
		JOIN st_user_group stug ON stug.User_ID=stu.ID
		JOIN st_group stg ON stg.ID=stug.Group_ID
		JOIN st_group_resource stgr ON stgr.Group_ID=stg.ID
		JOIN et_resource etr ON etr.ID=stgr.Resource_ID
		JOIN et_access_level etal ON etal.ID=stgr.Access_level_ID
		JOIN et_access_type etat ON etat.ID=stgr.Access_type_ID
	WHERE stu.ID=$1
		AND stu.User_status_ID=1
		AND ((UPPER(etr.Name)=UPPER($2)
		AND UPPER(etal.Name)=UPPER($3))
			 OR stg.admin_flag)
	GROUP BY etat.ID, stgr.admin_flag, stg.admin_flag
	LIMIT 1`
)

// CheckAccess Checks the user's access to the requested resource and stores the result of the check
// The query must return only the row based on the highest level of access the user is allowed.
// e.g. the order is Group, Owner, World
// Deny would take preferance over Allow - but we don't have this concept yet.
//
// ????? This may need to be rewritten and called from within each handler. ????????????
// for example...
// CheckAccess Checks that the user is authorised to take this action
// Resource = name of the data resource being accesses being accessed
// Action = type of access request action e.g. view, save, edit, list, delete
//
//	func UserCheckAccess(UserID, Resource string, Action string) bool {
//			// check that the user has permissions to take the requested action
//			// this might also consider information in record being accessed
//	}
func (h *Handler) UserCheckAccess(UserID int, ResourceName, ActionName string) (models.AccessCheck, error) {
	var err error
	var access models.AccessCheck

	err = h.appConf.Db.QueryRow(sqlUserCheckAccess, UserID, ResourceName, ActionName).Scan(&access.AccessTypeID, &access.AdminFlag)
	if err != nil { // If the number of rows returned is 0 then user is not authorised to access the resource
		log.Printf("%v %v %v %v %+v %v %v %v %v %v %v", debugTag+"CheckAccess()3 Access denied", "err =", err, "access =", access, "UserID =", UserID, "ResourceName =", ResourceName, "ActionName =", ActionName)
		return models.AccessCheck{}, errors.New(debugTag + "UserCheckAccess: access denied (" + err.Error() + ")")
	}
	return access, nil
}

func (h *Handler) UserReadQry(id int) (models.User, error) {
	record := models.User{}
	err := h.appConf.Db.Get(&record, "SELECT id, name, username, email FROM st_users WHERE id = $1", id)
	if err != nil {
		return models.User{}, err
	}
	return record, nil
}

func (h *Handler) UserInsertQry(record models.User) (int, error) {
	err := h.appConf.Db.QueryRow(
		"INSERT INTO st_users (name, username, email) VALUES ($1, $2, $3) RETURNING id",
		record.Name, record.Username, record.Email).Scan(&record.ID)
	return record.ID, err
}

func (h *Handler) UserUpdateQry(record models.User) error {
	_, err := h.appConf.Db.Exec("UPDATE st_users SET name = $1, username = $2, email = $3 WHERE id = $4",
		record.Name, record.Username, record.Email, record.ID)
	return err
}

func (h *Handler) UserWriteQry(record models.User) (int, error) {
	var err error
	err = h.appConf.Db.QueryRow(sqlTokenFind, record.ID).Scan(&record.ID) // Check to see if a record exists
	if err != nil {
		record.ID, err = h.UserInsertQry(record) //No Existing record found so we are okay to insert the new record
	} else {
		err = h.UserUpdateQry(record) //Existing record found so we are okay to update the record
	}
	if err != nil {
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"TokenWriteQry()7 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
		return 0, err
	}
	return record.ID, err
}

const (
	//Sets the user status
	sqlSetStatus = `UPDATE st_users SET User_status_ID=$1 WHERE ID=$2`
)

// SetStatusID sets the users account status
func (h *Handler) UserSetStatusID(userID int, status handlerUserAccountStatus.AccountStatus) error {
	var err error

	result, err := h.appConf.Db.Exec(sqlSetStatus, status, userID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserCheckRepo.CheckPassword()2 ", "err =", err, "result =", result)
		return err //update failed
	}
	return nil //nil error = users status updated
}

// Token queries

const (
	sqlTokenDelete = `DELETE FROM st_token WHERE ID=$1`
	sqlTokenFind   = `SELECT ID FROM st_token WHERE ID=$1`
	sqlTokenInsert = `INSERT INTO st_token (user_iD, name, host, token, token_valid_id, valid_from, valid_to) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	sqlTokenRead   = `SELECT c.id, c.user_id, c.name, c.host, c.token, sv.name AS valid, c.token_valid_ID, c.valid_from, c.valid_to FROM st_token c LEFT JOIN se_token_valid sv ON sv.ID=c.token_valid_ID WHERE c.ID=$1`
	sqlTokenUpdate = `UPDATE st_token SET user_id=$1, name=$2, host=$3, token=$4, token_valid_ID=$5, valid_from=$6, valid_to=$7 WHERE id=$8`
)

func (h *Handler) TokenInsertQry(record models.Token) (int, error) {
	err := h.appConf.Db.QueryRow(sqlTokenInsert,
		record.UserID, record.Name, record.Host, record.TokenStr, record.ValidID, record.ValidFrom, record.ValidTo).Scan(&record.ID)
	return record.ID, err
}

func (h *Handler) TokenUpdateQry(record models.Token) error {
	err := h.appConf.Db.QueryRow(sqlTokenUpdate,
		record.UserID, record.Name, record.Host, record.TokenStr, record.ValidID, record.ValidFrom, record.ValidTo).Scan(&record.ID)
	return err
}

func (h *Handler) TokenWriteQry(record models.Token) (int, error) {
	var err error
	err = h.appConf.Db.QueryRow(sqlTokenFind, record.ID).Scan(&record.ID) // Check to see if a record exists
	if err != nil {
		record.ID, err = h.TokenInsertQry(record) //No Existing record found so we are okay to insert the new record
	} else {
		err = h.TokenUpdateQry(record) //Existing record found so we are okay to update the record
	}
	if err != nil {
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"TokenWriteQry()7 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
		return 0, err
	}
	return record.ID, err
}

func (h *Handler) TokenDeleteQry(recordID int) error {
	_, err := h.appConf.Db.Exec(sqlTokenDelete, recordID)
	return err
}

const (
	sqlTokenCleanOld = `DELETE FROM st_token
				USING st_token st1
				WHERE st1.id = $1
					AND st1.user_id = st_token.user_id
					AND st1.name = st_token.name
					AND substring(st1.host FROM '^[^:]+') = substring(st_token.host FROM '^[^:]+')
					AND st_token.id <> $1;`

	sqlTokenCleanExpired = `DELETE FROM st_token st WHERE st.Valid_to < CURRENT_TIMESTAMP`
)

// CleanOld removes tokens older than the latest token
func (h *Handler) TokenCleanOld(recordID int) error {
	var err error
	result, err := h.appConf.Db.Exec(sqlTokenCleanOld, recordID)
	if err != nil { // Various errors including: no record found; key constraints
		log.Printf("%v %v %v %v %+v", debugTag+"TokenCleanOld()1", "recordID", recordID, "result =", result)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v %v %v\n", debugTag+"TokenCleanOld()2", "err =", err, "recordID", recordID, "result", result, "rowsAffected =", rowsAffected)
		return err
	}
	return nil
}

func (h *Handler) TokenCleanExpired() error {
	var err error
	result, err := h.appConf.Db.Exec(sqlTokenCleanExpired)
	if err != nil { // Various errors including: no record found; key constraints
		log.Printf("%v %v %v %v %+v\n", debugTag+"TokenCleanExpired()1", "err =", err, "result =", result)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %v\n", debugTag+"TokenCleanExpired()2", "err =", err, "result", result, "rowsAffected =", rowsAffected)
		return err
	}
	return nil
}

const (
	//Finds only valid cookies where the user account is current
	//if the user account is disabled or set to new it will not return the cookie
	//if the cookie is not valid it will not return the cookie.
	sqlFindSessionToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.token_valid_ID, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		JOIN st_users stu ON stu.ID=stt.User_ID
		LEFT JOIN et_token_valid ettv ON ettv.ID=stt.token_valid_ID
	WHERE stt.token=$1 AND stt.Name='session' AND stt.token_valid_ID=1 AND (stu.User_status_ID=1 OR stu.User_status_ID=0)`

	//Finds valid tokens where user account exists and the token name is the same as the name passed in
	sqlFindToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.token_valid_ID, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		JOIN st_users stu ON stu.ID=stt.User_ID
		LEFT JOIN et_token_valid ettv ON ettv.ID=stt.token_valid_ID
	WHERE stt.token=$1 AND stt.Name=$2 AND stt.token_valid_ID=1`
)

// FindSessionToken using the session cookie string find session cookie data in the DB and return the token item
// if the cookie is not found return the DB error
func (h *Handler) FindSessionToken(cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	//err = r.DBConn.QueryRow(sqlFindCookie, result.CookieStr).Scan(&result.ID, &result.UserID, &result.Name, &result.CookieStr, &result.Valid, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	err = h.appConf.Db.QueryRow(sqlFindSessionToken, result.TokenStr).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"FindSessionToken()2", "err =", err, "sqlFindSessionToken =", sqlFindSessionToken, "result =", result)
		return result, err
	}
	return result, nil
}

// FindToken using the session cookie name and cookie string find session cookie data in the DB and return the token item
func (h *Handler) FindToken(name, cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	result.Name.SetValid(name)
	err = h.appConf.Db.QueryRow(sqlFindToken, result.TokenStr, result.Name).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"FindToken()2", "err =", err, "sqlFindToken =", sqlFindToken, "result =", result)
		return result, err
	}
	return result, nil
}

const (
	//Finds records associated with a users access
	//no access is allowed if no records are found

	sqlAccessCheck = `SELECT DISTINCT eal.ID, eal.Name
	FROM st_user stu
	 JOIN st_user_group stug ON sug.User_ID=stu.ID
	 JOIN st_group_resource stgr ON sgr.Group_ID=sug.Group_ID
	 JOIN et_resource etr ON etr.ID=sgr.Resource_ID
	 JOIN et_access_level etal ON etal.ID = sgr.Access_level_ID 
	WHERE stu.ID=$1 AND etr.ID=$2`
)

func (h *Handler) AccessCheckXX(userID int, resourceID int, accessLevelID int) error {
	var err error
	var result int
	err = h.appConf.Db.QueryRow(sqlAccessCheck, userID, resourceID, accessLevelID).Scan(&result)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"AccessCheck1", "err =", err, "sqlFindToken =", sqlFindToken, "result =", result)
		return err
	}
	return nil
}
