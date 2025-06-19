package handlerAuthTemplate

import (
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/models"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"
)

const debugTag = "handlerAuthTemplate."

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

//(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest interface{}, query string)
//(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest interface{}, query string, args ...interface{})

func UserCheckAccess(debugStr string, Db *sqlx.DB, UserID int, ResourceName, ActionName string) (models.AccessCheck, error) {
	var err error
	var access models.AccessCheck

	err = Db.QueryRow(sqlUserCheckAccess, UserID, ResourceName, ActionName, handlerUserAccountStatus.AccountActive).Scan(&access.AccessTypeID, &access.AdminFlag)
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
func UserSetStatusID(debugStr string, Db *sqlx.DB, userID int, status handlerUserAccountStatus.AccountStatus) error {
	var err error

	result, err := Db.Exec(sqlSetStatus, status, userID)
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

func TokenInsertQry(debugStr string, Db *sqlx.DB, record models.Token) (int, error) {
	err := Db.QueryRow(sqlTokenInsert,
		record.UserID, record.Name, record.Host, record.TokenStr, record.Valid, record.ValidFrom, record.ValidTo).Scan(&record.ID)
	return record.ID, err
}

func TokenUpdateQry(debugStr string, Db *sqlx.DB, record models.Token) error {
	err := Db.QueryRow(sqlTokenUpdate,
		record.ID, record.UserID, record.Name, record.Host, record.TokenStr, record.Valid, record.ValidFrom, record.ValidTo).Scan(&record.ID)
	return err
}

func TokenWriteQry(debugStr string, Db *sqlx.DB, record models.Token) (int, error) {
	var err error
	err = Db.QueryRow(sqlTokenFind, record.ID).Scan(&record.ID) // Check to see if a record exists
	if err != nil {
		record.ID, err = TokenInsertQry(debugStr, Db, record) //No Existing record found so we are okay to insert the new record
	} else {
		err = TokenUpdateQry(debugStr, Db, record) //Existing record found so we are okay to update the record
	}
	if err != nil {
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"TokenWriteQry()7 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
		return 0, err
	}
	return record.ID, err
}

func TokenDeleteQry(debugStr string, Db *sqlx.DB, recordID int) error {
	_, err := Db.Exec(sqlTokenDelete, recordID)
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
func TokenCleanOld(debugStr string, Db *sqlx.DB, recordID int) error {
	var err error
	result, err := Db.Exec(sqlTokenCleanOld, recordID)
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

func TokenCleanExpired(debugStr string, Db *sqlx.DB) error {
	var err error
	result, err := Db.Exec(sqlTokenCleanExpired)
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
func FindSessionToken(debugStr string, Db *sqlx.DB, cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	//err = r.DBConn.QueryRow(sqlFindCookie, result.CookieStr).Scan(&result.ID, &result.UserID, &result.Name, &result.CookieStr, &result.Valid, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	err = Db.QueryRow(sqlFindSessionToken, result.TokenStr, handlerUserAccountStatus.AccountActive).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.Valid, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"FindSessionToken()2", "err =", err, "sqlFindSessionToken =", sqlFindSessionToken, "result =", result)
		return result, err
	}
	return result, nil
}

// FindToken using the session cookie name and cookie string find session cookie data in the DB and return the token item
func FindToken(debugStr string, Db *sqlx.DB, name, cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	result.Name.SetValid(name)
	err = Db.QueryRow(sqlFindToken, result.TokenStr, result.Name).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.Valid, &result.ValidFrom, &result.ValidTo)
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

func AccessCheckXX(debugStr string, Db *sqlx.DB, userID int, resourceID int, accessLevelID int) error {
	var err error
	var result int
	err = Db.QueryRow(sqlAccessCheck, userID, resourceID, accessLevelID).Scan(&result)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"AccessCheck1", "err =", err, "sqlFindToken =", sqlFindToken, "result =", result)
		return err
	}
	return nil
}

const (
	sqlUserFind     = `SELECT id, FROM st_users WHERE id = $1`
	sqlUserRead     = `SELECT id, name, username, email FROM st_users WHERE id = $1`
	sqlUserNameRead = `SELECT id, name, username, email FROM st_users WHERE username = $1`
	sqlUserInsert   = `INSERT INTO st_users (name, username, email) VALUES ($1, $2, $3) RETURNING id`
	sqlUserUpdate   = `UPDATE st_users SET name = $1, username = $2, email = $3 WHERE id = $4`
)

func UserReadQry(debugStr string, Db *sqlx.DB, id int) (models.User, error) {
	record := models.User{}
	err := Db.Get(&record, sqlUserRead, id)
	if err != nil {
		return models.User{}, err
	}
	return record, nil
}

func UserNameReadQry(debugStr string, Db *sqlx.DB, username string) (models.User, error) {
	record := models.User{}
	err := Db.Get(&record, sqlUserNameRead, username)
	if err != nil {
		return models.User{}, err
	}
	return record, nil
}

func UserInsertQry(debugStr string, Db *sqlx.DB, record models.User) (int, error) {
	err := Db.QueryRow(sqlUserInsert, record.Name, record.Username, record.Email).Scan(&record.ID)
	return record.ID, err
}

func UserUpdateQry(debugStr string, Db *sqlx.DB, record models.User) error {
	_, err := Db.Exec(sqlUserUpdate, record.Name, record.Username, record.Email, record.ID)
	return err
}

func UserWriteQry(debugStr string, Db *sqlx.DB, record models.User) (int, error) {
	var err error
	err = Db.QueryRow(sqlUserFind, record.ID).Scan(&record.ID) // Check to see if a record exists
	if err != nil {
		record.ID, err = UserInsertQry(debugStr, Db, record) //No Existing record found so we are okay to insert the new record
	} else {
		err = UserUpdateQry(debugStr, Db, record) //Existing record found so we are okay to update the record
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

// createSessionToken store it in the DB and in the session struct and return *http.Cookie
func CreateSessionToken(debugStr string, Db *sqlx.DB, userID int, host string) (*http.Cookie, error) {
	var err error
	//expiration := time.Now().Add(365 * 24 * time.Hour)
	sessionToken := &http.Cookie{
		Name:  "session",
		Value: uuid.NewV4().String(),
		Path:  "/",
		//Domain: "localhost",
		//Expires:    time.Time{},
		//RawExpires: "",
		//MaxAge:     0,
		//Secure:   false,
		Secure:   true,  //https --> true,
		HttpOnly: false, //https --> true, http --> false
		SameSite: http.SameSiteNoneMode,
		//SameSite: http.SameSiteLaxMode,
		//SameSite: http.SameSiteStrictMode,
		//Raw:        "",
		//Unparsed:   []string{},
	}
	// Store the session cookie for the user in the database
	tokenItem := models.Token{}
	tokenItem.UserID = userID
	tokenItem.Name.SetValid(sessionToken.Name)
	tokenItem.Host.SetValid(host)
	tokenItem.TokenStr.SetValid(sessionToken.Value)
	tokenItem.SessionData.SetValid("")
	tokenItem.Valid.SetValid(true)
	tokenItem.ValidFrom.SetValid(time.Now())
	tokenItem.ValidTo.SetValid(time.Now().Add(24 * time.Hour))

	tokenItem.ID, err = TokenWriteQry(debugStr, Db, tokenItem)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()1 Fatal: createSessionToken fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
	} else {
		err = TokenCleanOld(debugStr, Db, tokenItem.ID)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()2: Token CleanOld fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
		}
		TokenCleanExpired(debugStr, Db)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()3: Token CleanExpired fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
		}
	}
	return sessionToken, err
}
