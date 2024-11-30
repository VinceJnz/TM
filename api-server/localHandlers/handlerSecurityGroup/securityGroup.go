package handlerSecurityGroup

import (
	"encoding/json"
	"log"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerSecurityGroup."

const (
	qryGetAll = `SELECT id, name, description, admin_flag FROM st_group`
	qryGet    = `SELECT id, name, description, admin_flag FROM st_group WHERE id = $1`
	qryCreate = `INSERT INTO st_group (name, description, admin_flag) VALUES ($1, $2, $3) RETURNING id`
	qryUpdate = `UPDATE st_group SET name = $1, description = $2, admin_flag = $3 WHERE id = $4`
	qryDelete = `DELETE FROM st_group WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	handlerStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.Group{}, qryGetAll, nil)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.Group{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Group
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Name, record.Description, record.AdminFlag)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.Group
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.Name, record.Description, record.AdminFlag, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
