package dbAuthTemplate

import (
	"api-server/v2/models"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

const debugTag = "dbAuthTemplate."

// sqlCheckAccess checks that the user account has been activated and that it has access to the requested resource and method.
// NOTE: If the group (stg) admin_flag is set then access is given regardless of the resource or action settings.
const (
	sqlUserCheckAccess = `SELECT etat.ID, stg.Role
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
			 OR stg.Role IN ('admin', 'sysadmin'))
	GROUP BY etat.ID, stg.Role
	ORDER BY
		CASE LOWER(stg.Role::text)
			WHEN 'sysadmin' THEN 3
			WHEN 'admin' THEN 2
			WHEN 'user' THEN 1
			ELSE 0
		END DESC,
		etat.ID DESC
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
func UserCheckAccess(debugStr string, Db *sqlx.DB, UserID int, ResourceName, ActionName string) (models.AccessCheck, error) {
	var err error
	var access models.AccessCheck

	err = Db.QueryRow(sqlUserCheckAccess, UserID, ResourceName, ActionName, models.AccountActive).Scan(&access.AccessTypeID, &access.Role)
	if err != nil { // If the number of rows returned is 0 then user is not authorised to access the resource
		log.Printf("%v %v %v %v %+v %v %v %v %v %v %v", debugTag+"UserCheckAccess()3 Access denied", "err =", err, "access =", access, "UserID =", UserID, "ResourceName =", ResourceName, "ActionName =", ActionName)
		return models.AccessCheck{}, errors.New(debugTag + "UserCheckAccess: access denied (" + err.Error() + ")")
	}
	return access, nil
}

const (
	//Sets the user status
	sqlSetStatus = `UPDATE st_users SET user_account_status_id=$1 WHERE ID=$2`
)

// SetStatusID sets the users account status
func UserSetStatusID(debugStr string, Db *sqlx.DB, userID int, status models.AccountStatus) error {
	var err error

	result, err := Db.Exec(sqlSetStatus, status, userID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserSetStatusID()2 ", "err =", err, "result =", result)
		return err //update failed
	}
	return nil //nil error = users status updated
}

// Token queries

const (
	sqlTokenDelete = `DELETE FROM st_token WHERE ID=$1`
	sqlTokenFind   = `SELECT ID FROM st_token WHERE ID=$1`
	// Include session_data so arbitrary JSON payloads can be stored on tokens (e.g. registration data)
	sqlTokenInsert = `INSERT INTO st_token (user_id, name, host, token, session_data, token_valid, valid_from, valid_to) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	sqlTokenRead   = `SELECT c.id, c.user_id, c.name, c.host, c.token, c.session_data, c.token_valid, c.valid_from, c.valid_to FROM st_token c WHERE c.ID=$1`
	sqlTokenUpdate = `UPDATE st_token SET (user_id, name, host, token, session_data, token_valid, valid_from, valid_to) = ($2, $3, $4, $5, $6, $7, $8, $9) WHERE id=$1`
)

func TokenInsertQry(debugStr string, Db *sqlx.DB, record models.Token) (int, error) {
	err := Db.QueryRow(sqlTokenInsert,
		record.UserID, record.Name, record.Host, record.TokenStr, record.SessionData, record.Valid, record.ValidFrom, record.ValidTo).Scan(&record.ID)
	return record.ID, err
}

func TokenUpdateQry(debugStr string, Db *sqlx.DB, record models.Token) error {
	result, err := Db.Exec(sqlTokenUpdate,
		record.ID, record.UserID, record.Name, record.Host, record.TokenStr, record.SessionData, record.Valid, record.ValidFrom, record.ValidTo)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
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
	log.Printf("%v %v %v", debugTag+"TokenDeleteQry()1."+debugStr, "recordID =", recordID)
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
	log.Printf("%v %v %v %v %v %v %+v %v %v\n", debugTag+"TokenCleanOld()3", "err =", err, "recordID", recordID, "result", result, "rowsAffected =", rowsAffected)

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
	log.Printf("%v %v %v %v %+v %v %v\n", debugTag+"TokenCleanExpired()3", "err =", err, "result", result, "rowsAffected =", rowsAffected)
	return nil
}

const (
	//Finds only valid cookies where the user account is current
	//if the user account is disabled or set to new it will not return the cookie
	//if the cookie is not valid it will not return the cookie.
	sqlFindSessionToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.session_data, stt.token_valid, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		JOIN st_users stu ON stu.ID=stt.User_ID
	WHERE stt.token=$1 AND stt.Name='session' AND stt.token_valid AND stu.user_account_status_ID=$2 AND stt.Valid_From < CURRENT_TIMESTAMP AND stt.Valid_To > CURRENT_TIMESTAMP`
	//WHERE stt.token=$1 AND stt.Name='session' AND stt.token_valid AND stu.user_account_status_ID=$2 AND stt.Valid_To < CURRENT_TIMESTAMP AND stt.Valid_From > CURRENT_TIMESTAMP`

	//Finds valid tokens where user account exists and the token name is the same as the name passed in
	//Finds valid tokens where the token name is the same as the name passed in
	sqlFindToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.session_data, stt.token_valid, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		--JOIN st_users stu ON stu.ID=stt.User_ID
		WHERE stt.token=$1 AND stt.Name=$2 AND stt.token_valid AND stt.Valid_From < CURRENT_TIMESTAMP AND stt.Valid_To > CURRENT_TIMESTAMP`
	//WHERE stt.token=$1 AND stt.Name=$2 AND stt.token_valid AND stt.Valid_To < CURRENT_TIMESTAMP AND stt.Valid_From > CURRENT_TIMESTAMP`
)

// FindSessionToken using the session cookie string find session cookie data in the DB and return the token item
// if the cookie is not found return the DB error
func FindSessionToken(debugStr string, Db *sqlx.DB, cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	//err = r.DBConn.QueryRow(sqlFindCookie, result.CookieStr).Scan(&result.ID, &result.UserID, &result.Name, &result.CookieStr, &result.Valid, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	err = Db.QueryRow(sqlFindSessionToken, result.TokenStr, models.AccountActive).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.SessionData, &result.Valid, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		//log.Printf("%v %v %v %v %v %v %+v", debugTag+"FindSessionToken()2", "err =", err, "sqlFindSessionToken =", sqlFindSessionToken, "result =", result)
		return result, err
	}
	err = TokenCleanExpired(debugStr, Db)
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Handler.FindSessionToken()3: Token CleanExpired fail", "err =", err)
	}
	return result, nil
}

// FindToken using the session cookie name and cookie string find session cookie data in the DB and return the token item
func FindToken(debugStr string, Db *sqlx.DB, name, cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	//result.TokenStr.SetValid(cookieStr)
	//result.Name.SetValid(name)
	err = Db.QueryRow(sqlFindToken, cookieStr, name).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.SessionData, &result.Valid, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"FindToken()2", "err =", err, "sqlFindToken =", sqlFindToken, "result =", result)
		return result, err
	}
	err = TokenCleanExpired(debugStr, Db)
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Handler.FindToken()3: Token CleanExpired fail", "err =", err)
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
	sqlUserFind        = `SELECT id FROM st_users WHERE id = $1`
	sqlUserRead        = `SELECT id, name, username, email, user_address, user_birth_date, user_account_status_id, user_password, user_account_hidden, provider, provider_id FROM st_users WHERE id = $1`
	sqlUserNameRead    = `SELECT id, name, username, email, user_address, user_birth_date, user_account_status_id, user_password, user_account_hidden, provider, provider_id FROM st_users WHERE username = $1`
	sqlUserEmailRead   = `SELECT id, name, username, email, user_address, user_birth_date, user_account_status_id, user_password, user_account_hidden, provider, provider_id FROM st_users WHERE email = $1`
	sqlUserInsert      = `INSERT INTO st_users (name, username, email, user_address, user_birth_date, user_password, user_account_status_id, user_account_hidden, provider, provider_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	sqlUserUpdate      = `UPDATE st_users SET name = $1, username = $2, email = $3, user_address = $4, user_birth_date = $5, user_password = $6, user_account_status_id = $7, user_account_hidden = $8, provider = $9, provider_id = $10 WHERE id = $11`
	sqlUserDelProvider = `UPDATE st_users SET provider = null , provider_id = null WHERE id = $1`
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

func UserEmailReadQry(debugStr string, Db *sqlx.DB, email string) (models.User, error) {
	record := models.User{}
	err := Db.Get(&record, sqlUserEmailRead, email)
	if err != nil {
		return models.User{}, err
	}
	return record, nil
}

func UserWriteQry(debugStr string, Db *sqlx.DB, record models.User) (int, error) {
	var err error
	Tx, err := Db.Beginx() // Start a transaction
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserWriteQry()1 - ", "err =", err, "record =", record)
		return 0, err // If we can't start a transaction then we can't write the record
	}
	defer Tx.Rollback() // Ensure the transaction is rolled back if we don't commit it

	record.ID, err = UserWriteQryTx(debugStr, Tx, record) // Write the user record to the database
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserWriteQry()2 - ", "err =", err, "record =", record)
		return 0, err // If we can't write the record then we can't commit the transaction
	}
	err = Tx.Commit() // Commit the transaction
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserWriteQry()3 - ", "err =", err, "record =", record)
		return 0, err // If we can't commit the transaction then we can't write the record
	}
	return record.ID, err
}

func UserInsertQryTx(debugStr string, Db *sqlx.Tx, record models.User) (int, error) {
	log.Printf("%v %v %+v", debugTag+debugStr+"UserInsertQryTx()1 - ", "record =", record)
	err := Db.QueryRow(sqlUserInsert, record.Name, record.Username, record.Email, record.Address, record.BirthDate, record.Password, record.AccountStatusID, record.AccountHidden, record.Provider, record.ProviderID).Scan(&record.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserInsertQryTx()1 - ", "err =", err, "record =", record)
		return 0, err // If we can't commit the transaction then we can't write the record
	}
	return record.ID, err
}

func UserUpdateQryTx(debugStr string, Db *sqlx.Tx, record models.User) error {
	_, err := Db.Exec(sqlUserUpdate, record.Name, record.Username, record.Email, record.Address, record.BirthDate, record.Password, record.AccountStatusID, record.AccountHidden, record.Provider, record.ProviderID, record.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserUpdateQryTx()1 - ", "err =", err, "record =", record)
	}
	return err
}

func UserWriteQryTx(debugStr string, Db *sqlx.Tx, record models.User) (int, error) {
	var err error
	err = Db.QueryRow(sqlUserFind, record.ID).Scan(&record.ID) // Check to see if a record exists
	switch err {
	case sql.ErrNoRows:
		record.ID, err = UserInsertQryTx(debugStr, Db, record) //No Existing record found so we are okay to insert the new record
		if err != nil {
			log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"UserWriteQryTx()5 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
			return 0, err
		}
	case nil:
		err = UserUpdateQryTx(debugStr, Db, record) //Existing record found so we are okay to update the record
		if err != nil {
			log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"UserWriteQryTx()6 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
			return 0, err
		}
	default:
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"UserWriteQryTx()7 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
		return 0, err
	}
	return record.ID, err
}

func UserDelProvider(debugStr string, Db *sqlx.DB, id int) error {
	_, err := Db.Exec(sqlUserDelProvider, id)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"UserDelProvider()1 - ", "err =", err, "id =", id)
	}
	return err
}

// CreateSessionToken store it in the DB in the session struct and return *http.Cookie
func CreateSessionToken(debugStr string, Db *sqlx.DB, userID int, host string, expiration time.Time) (*http.Cookie, error) {
	var err error
	sessionToken, err := CreateNamedToken(debugStr, Db, true, userID, host, "session", expiration)
	return sessionToken, err
}

// CreateNamedToken and return *http.Token, and store it in the DB if storeToken is true
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Cookies
func CreateNamedToken(debugStr string, Db *sqlx.DB, storeToken bool, userID int, host, name string, expiration time.Time) (*http.Cookie, error) {
	var err error
	if name == "" {
		name = "temp_session_token"
	}
	//if expiration.IsZero() {
	//	expiration = time.Now().Add(3 * time.Minute) // Token valid for 3 minutes
	//}
	sessionToken := &http.Cookie{
		Name:    name,
		Value:   GenerateSecureToken(),
		Path:    "/",
		Domain:  host,
		Expires: expiration, // Session cookies — cookies without a Max-Age or Expires attribute – are deleted when the current session ends.
		//RawExpires: "",
		//MaxAge:     0,
		Secure:   true,  // A cookie with the Secure attribute is only sent to the server with an encrypted request over the HTTPS protocol
		HttpOnly: false, //https --> true, http --> false // A cookie with the HttpOnly attribute can't be accessed by JavaScript
		SameSite: http.SameSiteNoneMode,
		//SameSite: http.SameSiteLaxMode,
		//SameSite: http.SameSiteStrictMode, //Strict causes the browser to only send the cookie in response to requests originating from the cookie's origin site
		//Raw:        "",
		//Unparsed:   []string{},
	}

	tokenItem := models.Token{}
	if storeToken {
		// Store the session cookie for the user in the database
		tokenItem.UserID = userID
		tokenItem.Name.SetValid(sessionToken.Name)
		tokenItem.Host.SetValid(host)
		tokenItem.TokenStr.SetValid(sessionToken.Value)
		//tokenItem.SessionData.SetValid("") // Not yet implemented. This can be used to store session data in JSON format.
		tokenItem.Valid.SetValid(true)
		tokenItem.ValidFrom.SetValid(time.Now())
		validTo := expiration
		if validTo.IsZero() {
			validTo = time.Now().Add(1 * time.Hour)
		}
		tokenItem.ValidTo.SetValid(validTo)

		tokenItem.ID, err = TokenWriteQry(debugStr, Db, tokenItem)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v", debugTag+"CreateNamedToken()1 Fatal: createSessionToken fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
		} else {
			err = TokenCleanOld(debugStr, Db, tokenItem.ID)
			if err != nil {
				log.Printf("%v %v %v %v %v %v %+v", debugTag+"CreateNamedToken()2: Token CleanOld fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
			}
			err = TokenCleanExpired(debugTag+"CreateNamedToken()3 ", Db) // Clean expired tokens for the user
			if err != nil {
				log.Printf("%v %v %v %v %v %v %+v", debugTag+"CreateNamedToken()4: Token CleanExpired fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
			}
		}
	}
	log.Printf("%v %v %v %v %v %v %v %+v %+v", debugTag+"CreateNamedToken()5: Success, can advise client", "err =", err, "UserID =", userID, "sessionToken =", *sessionToken, "tokenItem =", tokenItem)
	return sessionToken, err
}
