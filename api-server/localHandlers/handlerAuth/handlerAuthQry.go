package handlerAuth

import (
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/models"
	"errors"
	"log"
	"math/big"
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
		AND stu.user_account_status_id=$4
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

	err = h.appConf.Db.QueryRow(sqlUserCheckAccess, UserID, ResourceName, ActionName, handlerUserAccountStatus.AccountActive).Scan(&access.AccessTypeID, &access.AdminFlag)
	if err != nil { // If the number of rows returned is 0 then user is not authorised to access the resource
		log.Printf("%v %v %v %v %+v %v %v %v %v %v %v", debugTag+"CheckAccess()3 Access denied", "err =", err, "access =", access, "UserID =", UserID, "ResourceName =", ResourceName, "ActionName =", ActionName)
		return models.AccessCheck{}, errors.New(debugTag + "UserCheckAccess: access denied (" + err.Error() + ")")
	}
	return access, nil
}

const (
	//Sets the user status
	sqlSetStatus = `UPDATE st_users SET user_account_status_id=$1 WHERE ID=$2`
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
	sqlTokenInsert = `INSERT INTO st_token (user_iD, name, host, token, token_valid, valid_from, valid_to) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	sqlTokenRead   = `SELECT c.id, c.user_id, c.name, c.host, c.token, c.token_valid, c.valid_from, c.valid_to FROM st_token c WHERE c.ID=$1`
	sqlTokenUpdate = `UPDATE st_token SET (user_id, name, host, token, token_valid, valid_from, valid_to) = ($2, $3, $4, $5, $6, $7, $8) WHERE id=$1`
)

func (h *Handler) TokenInsertQry(record models.Token) (int, error) {
	err := h.appConf.Db.QueryRow(sqlTokenInsert,
		record.UserID, record.Name, record.Host, record.TokenStr, record.Valid, record.ValidFrom, record.ValidTo).Scan(&record.ID)
	return record.ID, err
}

func (h *Handler) TokenUpdateQry(record models.Token) error {
	err := h.appConf.Db.QueryRow(sqlTokenUpdate,
		record.ID, record.UserID, record.Name, record.Host, record.TokenStr, record.Valid, record.ValidFrom, record.ValidTo).Scan(&record.ID)
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
	sqlFindSessionToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.token_valid, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		JOIN st_users stu ON stu.ID=stt.User_ID
	WHERE stt.token=$1 AND stt.Name='session' AND stt.token_valid AND stu.user_account_status_ID=$2`

	//Finds valid tokens where user account exists and the token name is the same as the name passed in
	sqlFindToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.token_valid, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		JOIN st_users stu ON stu.ID=stt.User_ID
	WHERE stt.token=$1 AND stt.Name=$2 AND stt.token_valid`
)

// FindSessionToken using the session cookie string find session cookie data in the DB and return the token item
// if the cookie is not found return the DB error
func (h *Handler) FindSessionToken(cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	//err = r.DBConn.QueryRow(sqlFindCookie, result.CookieStr).Scan(&result.ID, &result.UserID, &result.Name, &result.CookieStr, &result.Valid, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	err = h.appConf.Db.QueryRow(sqlFindSessionToken, result.TokenStr, handlerUserAccountStatus.AccountActive).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.Valid, &result.ValidFrom, &result.ValidTo)
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
	err = h.appConf.Db.QueryRow(sqlFindToken, result.TokenStr, result.Name).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.Valid, &result.ValidFrom, &result.ValidTo)
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

const (
	sqlUserFind   = `SELECT id, FROM st_users WHERE id = $1`
	sqlUserRead   = `SELECT id, name, username, email FROM st_users WHERE id = $1`
	sqlUserInsert = `INSERT INTO st_users (name, username, email) VALUES ($1, $2, $3) RETURNING id`
	sqlUserUpdate = `UPDATE st_users SET name = $1, username = $2, email = $3 WHERE id = $4`
)

func (h *Handler) UserReadQry(id int) (models.User, error) {
	record := models.User{}
	err := h.appConf.Db.Get(&record, sqlUserRead, id)
	if err != nil {
		return models.User{}, err
	}
	return record, nil
}

func (h *Handler) UserInsertQry(record models.User) (int, error) {
	err := h.appConf.Db.QueryRow(sqlUserInsert, record.Name, record.Username, record.Email).Scan(&record.ID)
	return record.ID, err
}

func (h *Handler) UserUpdateQry(record models.User) error {
	_, err := h.appConf.Db.Exec(sqlUserUpdate, record.Name, record.Username, record.Email, record.ID)
	return err
}

func (h *Handler) UserWriteQry(record models.User) (int, error) {
	var err error
	err = h.appConf.Db.QueryRow(sqlUserFind, record.ID).Scan(&record.ID) // Check to see if a record exists
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

// UserCheckRepo ??
//type UtilsRepo struct {
//	DBConn *dataStore.DB
//}

const (
	//Returns a result if the username/password combination is correct and the user account is current.
	//sqlGetUserSalt   = `SELECT ID, User_name, Salt FROM st_user WHERE User_name=? and user_account_status_id=1`
	sqlGetUserSalt = `SELECT ID, username, user_account_status_id, salt FROM st_users WHERE username=$1` //The controller needs to check the user_account_status_id to make sure the auth process should proceed.
	//sqlCheckUserAuth = `SELECT ID, User_name, Salt FROM st_user WHERE User_name=? and Password=? and user_account_status_id=1`
	sqlCheckUserAuth = `SELECT ID, username, Salt FROM st_users WHERE username=$1 and user_account_status_id=$2`
	//sqlPutUserAuth   = `UPDATE st_user SET Password=?, Salt=?, Verifier=? WHERE ID=?`
	sqlUserAuthUpdate = `UPDATE st_users SET salt=$1, verifier=$2 WHERE id=$3`
	//sqlGetUserAuth   = `SELECT ID, User_name, Email, Salt, Verifier FROM st_user WHERE User_name=? and user_account_status_id=1`
	sqlGetUserAuth  = `SELECT ID, username, name, email, user_account_status_id, salt, verifier FROM st_users WHERE username=$1` //The controller needs to check the user_account_status_id to make sure the auth process should proceed.
	sqlGetAdminList = `SELECT su.ID, su.username, su.name, su.email
		FROM st_users su JOIN st_user_group sug ON sug.user_id = su.id
		WHERE su.user_account_status_id=$2 AND sug.group_id=$1`
)

// CheckPassword checks that the users Auth info matches the Auth info stored in the DB
// Need to depreciate this ??????????????? why??????????????? it is currently used by ctrlMainLogin
func (h *Handler) CheckUserAuthXX(username, password string) (models.User, error) {
	var err error
	var result models.User

	//err = r.DBConn.QueryRow(sqlCheckUserAuth, username, password).Scan(&result.ID, &result.UserName, &result.Salt)
	err = h.appConf.Db.QueryRow(sqlCheckUserAuth, username, handlerUserAccountStatus.AccountActive).Scan(&result.ID, &result.Username, &result.Salt)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"CheckPassword()2 ", "err =", err, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return models.User{}, err //password check failed
	}
	//return result, nil //nil error = password check succeeded, return User.ID, User.UserName
	return result, errors.New("Depreciated")
}

// PutUserAuth stores the user auth info in the user table
// Need to depreciate this ??????????????? why??????????????? it is currently used in by the following...
// handler\rest\ctrlAuth\ctrlAuthRegister.go:107:14: h.srvc.PutUserAuth undefined (type *store.Service has no field or method PutUserAuth)
// handler\rest\ctrlAuth\ctrlAuthReset.go:127:14: h.srvc.PutUserAuth undefined (type *store.Service has no field or method PutUserAuth)
func (h *Handler) UserAuthUpdate(user models.User) error {
	var err error
	v, err := user.Verifier.GobEncode() //Might need to do the encoding here
	log.Printf("%v %v %+v %v %+v %v %+v", debugTag+"UserAuthUpdate()1 ", "user =", user, "err =", err, "v =", v)

	result, err := h.appConf.Db.Exec(sqlUserAuthUpdate, user.Salt, v, user.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"UserAuthUpdate()2 ", "err =", err, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return err //Auth set failed
	}
	return nil //Auth set succeeded
	//return errors.New("Depreciated")
}

// GetUserAuth retrieves the user auth info from the user table
func (h *Handler) GetUserAuth(username string) (models.User, error) {
	var err error
	//var result User
	var v []byte
	result := &models.User{
		Verifier: &big.Int{},
	}

	err = h.appConf.Db.QueryRow(sqlGetUserAuth, username).Scan(&result.ID, &result.Username, &result.Name, &result.Email, &result.AccountStatusID, &result.Salt, &v) //??????? Validator might fail, so we will need to post process it ???????
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"GetUserAuth()2 ", "err =", err, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return models.User{}, err //get UserAuth failed
	}

	err = result.Verifier.GobDecode(v) //Decode the verifier
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"GetUserAuth()3 ", "err =", err, "result =", result, "v =", v)
		return models.User{}, err //get UserAuth failed
	}

	//log.Printf("%v %v %v %v %+v %v %+v", debugTag+"GetUserAuth()4 ", "err =", err, "result =", result, "v =", v)

	return *result, nil //return UserAuth info
}

// GetUserSalt gets the Auth salt for the specified username
func (h *Handler) GetUserSalt(username string) (models.User, error) {
	var err error
	var result models.User

	err = h.appConf.Db.QueryRow(sqlGetUserSalt, username).Scan(&result.ID, &result.Username, &result.AccountStatusID, &result.Salt)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v %v %+v %v %+v", debugTag+"GetUserSalt()2 ", "err =", err, "username =", username, "sqlGetUserSalt =", sqlGetUserSalt, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return models.User{}, err //GetSalt failed
	}
	return result, nil //return User.ID, User.UserName, User.Salt
}

// GetAdminList retrieves the user details (email address, etc) for the specified group
func (h *Handler) GetAdminList(groupID int64) ([]models.User, error) {
	var err error
	var result models.User
	var list []models.User

	rows, err := h.appConf.Db.Query(sqlGetAdminList, groupID, handlerUserAccountStatus.AccountActive)
	if err != nil {
		log.Println(debugTag+"GetAdminList()1 - ", "groupID =", groupID, "sqlGetAdminList =", sqlGetAdminList, "err =", err)
		//log.Fatal(err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&result.ID, &result.Username, &result.Name, &result.Email)
		if err != nil {
			log.Println(debugTag+"GetAdminList()2 - ", "sqlGetAdminList =", sqlGetAdminList, "result =", result, "err =", err)
			//log.Fatal(err)
			return nil, err
		}
		list = append(list, result) //append each result to the results array.
	}
	return list, nil
}
