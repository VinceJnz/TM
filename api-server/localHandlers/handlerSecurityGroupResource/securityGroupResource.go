package handlerSecurityGroupResource

import (
	"encoding/json"
	"log"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/dbTemplates/dbStandardTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerSecurityGroupResource."

const (
	qryGetAll = `SELECT stgr.id, stgr.group_id, stg.name AS group_name, resource_id, etr.name AS resource, access_level_id, etal.name AS access_level, access_type_id, etat.name AS access_type, stgr.admin_flag 
					FROM st_group_resource stgr
						JOIN st_group stg ON stg.id=stgr.group_id
						JOIN et_resource etr ON etr.id=stgr.resource_id
						JOIN et_access_level etal ON etal.id=stgr.access_level_id
						JOIN et_access_type etat ON etat.id=stgr.access_type_id`
	qryGet    = `SELECT id, group_id, resource_id, access_level_id, access_type_id, admin_flag FROM st_group_resource WHERE id = $1`
	qryCreate = `INSERT INTO st_group_resource (group_id, resource_id, access_level_id, access_type_id, admin_flag) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	qryUpdate = `UPDATE st_group_resource SET group_id = $1, resource_id = $2, access_level_id = $3, access_type_id = $4, admin_flag = $5 WHERE id = $6`
	qryDelete = `DELETE FROM st_group_resource WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.GroupResource{}, qryGetAll)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.GroupResource{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.GroupResource
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.GroupID, record.ResourceID, record.AccessLevelID, record.AccessTypeID, record.AdminFlag)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.GroupResource
	id := dbStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.GroupID, record.ResourceID, record.AccessLevelID, record.AccessTypeID, record.AdminFlag, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
