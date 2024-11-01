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
