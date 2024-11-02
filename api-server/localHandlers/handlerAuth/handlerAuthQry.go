package handlerAuth

import (
	"api-server/v2/models"
	"log"
)

// sqlCheckAccess checks that the user account has been activated and that it has access to the requested resource and method.
const (
	sqlUserCheckAccess = `SELECT eat.ID
	FROM st_user su
		JOIN st_user_group sug ON sug.User_ID=su.ID
		JOIN st_group sg ON sg.ID=sug.Group_ID
		JOIN st_group_resource sgr ON sgr.Group_ID=sg.ID
		JOIN se_resource er ON er.ID=sgr.Resource_ID
		JOIN se_access_level eal ON eal.ID=sgr.Access_level_ID
		JOIN se_access_type eat ON eat.ID=sgr.Access_type_ID
	WHERE su.ID=$1
		AND su.User_status_ID=1
		AND er.Name=$2
		AND eal.Name=$3
	GROUP BY eat.ID
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
//	func CheckAccess(UserID, Resource string, Action string) bool {
//			// check that the user has permissions to take the requested action
//			// this might also consider information in record being accessed
//	}
func (h *Handler) UserCheckAccess(UserID int, ResourceName, ActionName string) (int, error) {
	var err error
	var accessType int
	err = h.appConf.Db.QueryRow(sqlUserCheckAccess, UserID, ResourceName, ActionName).Scan(&accessType)
	if err != nil { // If the number of rows returned is 0 then user is not authorised to access the resource
		log.Println(debugTag+"AccessRepo.CheckAccess()3 ", "Access denied", "err =", err, "accessType =", accessType, "UserID =", UserID, "ResourceName =", ResourceName, "ActionName =", ActionName)
		return 0, err
	}
	return accessType, nil
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
	sqlSetStatus = `UPDATE st_user SET User_status_ID=$1 WHERE ID=$2`
)

// SetStatusID sets the users account status
func (h *Handler) UserSetStatusID(userID int, status models.AccountStatus) error {
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
		record.UserID, record.Name, record.Host, record.TokenStr, record.ValidID, record.ValidFrom, record.ValidTo, record.ID).Scan(record.ID)
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
	_, err := h.appConf.Db.Exec("DELETE FROM st_users WHERE id = $1", recordID)
	return err
}

const (
	//	sqlTokenCleanOld = `DELETE st1
	//FROM st_token st
	//	JOIN st_token st1 ON st1.User_ID = st.User_ID
	//								AND st1.Name = st.Name
	//								AND substring(st1.host FROM '.+]') = substring(st.host FROM '.+]')
	//								AND st1.ID <> st.ID
	//WHERE st.ID = $1`

	sqlTokenCleanOld = `DELETE FROM st_token st
		USING st_token st1
		WHERE st1.ID = $1
			AND st1.User_ID = st.User_ID
			AND st1.Name = st.Name
			AND substring(st1.host FROM '.+]') = substring(st.host FROM '.+]')
			AND st.id <> $1`

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
	sqlFindSessionToken = `SELECT c.ID, c.User_ID, c.Name, c.token, c.token_valid_ID, c.Valid_From, c.Valid_To
	FROM st_token c
		JOIN st_user u ON u.ID=c.User_ID
		LEFT JOIN se_token_valid sv ON sv.ID=c.token_valid_ID
	WHERE c.token=$1 AND c.Name='session' AND c.token_valid_ID=1 AND u.User_status_ID=1`

	//Finds valid tokens where user account exists and the token name is the same as the name passed in
	sqlFindToken = `SELECT c.ID, c.User_ID, c.Name, c.token, c.token_valid_ID, c.Valid_From, c.Valid_To
	FROM st_token c
		JOIN st_user u ON u.ID=c.User_ID
		LEFT JOIN se_token_valid sv ON sv.ID=c.token_valid_ID
	WHERE c.token=$1 AND c.Name=$2 AND c.token_valid_ID=1`
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
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"SessionRepo.FindSessionToken()2", "err =", err, "sqlFindSessionToken =", sqlFindSessionToken, "result =", result)
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
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"SessionRepo.FindToken()2", "err =", err, "sqlFindToken =", sqlFindToken, "result =", result)
		return result, err
	}
	return result, nil
}
