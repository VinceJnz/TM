package handlerSecurityGroup

import (
	"api-server/v2/models"
)

func (h *Handler) GetAllQry() ([]models.Group, error) {
	records := []models.Group{}
	err := h.appConf.Db.Select(&records, `SELECT id, name, description, admin_flag FROM st_group`)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (h *Handler) GetQry(id int) (models.Group, error) {
	record := models.Group{}
	err := h.appConf.Db.Get(&record, "SELECT id, name, description, admin_flag FROM st_group WHERE id = $1", id)
	if err != nil {
		return models.Group{}, err
	}
	return record, nil
}

func (h *Handler) CreateQry(record models.Group) error {
	err := h.appConf.Db.QueryRow(
		"INSERT INTO st_group (name, description, admin_flag) VALUES ($1, $2, $3) RETURNING id",
		record.Name, record.Description, record.AdminFlag).Scan(&record.ID)
	return err
}

func (h *Handler) UpdateQry(record models.Group) error {
	_, err := h.appConf.Db.Exec("UPDATE st_group SET name = $1, description = $2, admin_flag = $3 WHERE id = $4",
		record.Name, record.Description, record.AdminFlag, record.ID)
	return err
}

func (h *Handler) DeleteQry(recordID int) error {
	_, err := h.appConf.Db.Exec("DELETE FROM st_group WHERE id = $1", recordID)
	return err
}
