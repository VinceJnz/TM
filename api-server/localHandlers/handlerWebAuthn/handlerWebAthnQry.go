package handlerWebAuthn

import (
	"api-server/v2/models"
	"log"
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
