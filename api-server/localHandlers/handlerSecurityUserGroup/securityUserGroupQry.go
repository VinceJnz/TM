package handlerSecurityUserGroup

import (
	"api-server/v2/models"
)

func (h *Handler) GetAllQry() ([]models.UserGroup, error) {
	records := []models.UserGroup{}
	err := h.appConf.Db.Select(&records, `SELECT id, user_id, group_id FROM st_user_group`)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (h *Handler) GetQry(id int) (models.UserGroup, error) {
	record := models.UserGroup{}
	err := h.appConf.Db.Get(&record, "SELECT id, user_id, group_id FROM st_user_group WHERE id = $1", id)
	if err != nil {
		return models.UserGroup{}, err
	}
	return record, nil
}

func (h *Handler) CreateQry(record models.UserGroup) error {
	err := h.appConf.Db.QueryRow(
		"INSERT INTO st_user_group (user_id, group_id) VALUES ($1, $2) RETURNING id",
		record.UserID, record.GroupID).Scan(&record.ID)
	return err
}

func (h *Handler) UpdateQry(record models.UserGroup) error {
	_, err := h.appConf.Db.Exec("UPDATE st_user_group SET user_id = $1, group_id = $2 WHERE id = $3",
		record.UserID, record.GroupID, record.ID)
	return err
}

func (h *Handler) DeleteQry(recordID int) error {
	_, err := h.appConf.Db.Exec("DELETE FROM st_user_group WHERE id = $1", recordID)
	return err
}
