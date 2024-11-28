package handlerSecurityGroupResource

import (
	"api-server/v2/models"
)

const (
	GetAll = `SELECT stgr.id, group_id, resource_id, etr.name AS resource, access_level_id, etal.name AS access_level, access_type_id, etat.name AS access_type, stgr.admin_flag 
	FROM st_group_resource stgr
		JOIN st_group stg ON stg.id=stgr.group_id
		JOIN et_resource etr ON etr.id=stgr.resource_id
		JOIN et_access_level etal ON etal.id=stgr.access_level_id
		JOIN et_access_type etat ON etat.id=stgr.access_type_id`
)

func (h *Handler) GetAllQry() ([]models.GroupResource, error) {
	records := []models.GroupResource{}
	err := h.appConf.Db.Select(&records, GetAll)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (h *Handler) GetQry(id int) (models.GroupResource, error) {
	record := models.GroupResource{}
	err := h.appConf.Db.Get(&record, "SELECT id, group_id, resource_id, access_level_id, access_type_id, admin_flag FROM st_group_resource WHERE id = $1", id)
	if err != nil {
		return models.GroupResource{}, err
	}
	return record, nil
}

func (h *Handler) CreateQry(record models.GroupResource) error {
	err := h.appConf.Db.QueryRow(
		"INSERT INTO st_group_resource (group_id, resource_id, access_level_id, access_type_id, admin_flag) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		record.GroupID, record.ResourceID, record.AccessLevelID, record.AccessTypeID, record.AdminFlag).Scan(&record.ID)
	return err
}

func (h *Handler) UpdateQry(record models.GroupResource) error {
	_, err := h.appConf.Db.Exec("UPDATE st_group_resource SET group_id = $1, resource_id = $2, access_level_id = $3, access_type_id = $4, admin_flag = $5 WHERE id = $6",
		record.GroupID, record.ResourceID, record.AccessLevelID, record.AccessTypeID, record.AdminFlag, record.ID)
	return err
}

func (h *Handler) DeleteQry(recordID int) error {
	_, err := h.appConf.Db.Exec("DELETE FROM st_group_resource WHERE id = $1", recordID)
	return err
}
