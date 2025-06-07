package handlerWebAuthn

import (
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/models"
	"log"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
)

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

// createSessionToken store it in the DB and in the session struct and return *http.Cookie
func (h *Handler) createSessionToken(userID int, host string) (*http.Cookie, error) {
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

	tokenItem.ID, err = h.TokenWriteQry(tokenItem)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()1 Fatal: createSessionToken fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
	} else {
		err = h.TokenCleanOld(tokenItem.ID)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()2: Token CleanOld fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
		}
		h.TokenCleanExpired()
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()3: Token CleanExpired fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
		}
	}
	return sessionToken, err
}
