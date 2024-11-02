package handlerUser

import (
	"api-server/v2/models"
)

func (h *Handler) GetAllQry() (*[]models.User, error) {
	records := &[]models.User{}
	err := h.appConf.Db.Select(&records, `SELECT id, name, username, email FROM st_users`)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (h *Handler) GetQry(id int) (*models.User, error) {
	record := &models.User{}
	err := h.appConf.Db.Get(&record, "SELECT id, name, username, email FROM st_users WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (h *Handler) CreateQry(record models.User) error {
	err := h.appConf.Db.QueryRow(
		"INSERT INTO st_users (name, username, email) VALUES ($1, $2, $3) RETURNING id",
		record.Name, record.Username, record.Email).Scan(&record.ID)
	return err
}

func (h *Handler) UpdateQry(record models.User) error {
	_, err := h.appConf.Db.Exec("UPDATE st_users SET name = $1, username = $2, email = $3 WHERE id = $4",
		record.Name, record.Username, record.Email, record.ID)
	return err
}

func (h *Handler) DeleteQry(recordID int) error {
	_, err := h.appConf.Db.Exec("DELETE FROM st_users WHERE id = $1", recordID)
	return err
}
