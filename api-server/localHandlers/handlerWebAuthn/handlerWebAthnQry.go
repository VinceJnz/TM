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
