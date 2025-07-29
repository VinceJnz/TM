package handlerSecurityUserGroup

import (
	"encoding/json"
	"log"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/dbTemplates/handlerStandardTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerSecurityGroupResource."

const (
	qryGetAll = `SELECT stug.id, stug.user_id, stu.name as user_name, stug.group_id, stg.name as group_name
					FROM st_user_group stug
						JOIN st_users stu ON stu.id=stug.user_id
						JOIN st_group stg ON stg.id=stug.group_id`
	qryGet    = `SELECT id, user_id, group_id FROM st_user_group WHERE id = $1`
	qryCreate = `INSERT INTO st_user_group (user_id, group_id) VALUES ($1, $2) RETURNING id`
	qryUpdate = `UPDATE st_user_group SET user_id = $1, group_id = $2 WHERE id = $3`
	qryDelete = `DELETE FROM st_user_group WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	handlerStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.UserGroup{}, qryGetAll)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.UserGroup{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.UserGroup
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.UserID, record.GroupID)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.UserGroup
	id := handlerStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.UserID, record.GroupID, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
