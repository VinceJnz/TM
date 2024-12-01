package handlerUser

import (
	"encoding/json"
	"log"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerUser."

const (
	//qryGetAll = `SELECT id, name, username, email FROM st_users`
	qryGetAll = `SELECT id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, user_status_id, user_account_status_id, created, modified FROM st_users`
	//qryGet    = `SELECT id, name, username, email FROM st_users WHERE id = $1`
	qryGet    = `SELECT id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, user_status_id, user_account_status_id, created, modified FROM st_users WHERE id = $1`
	qryCreate = `INSERT INTO st_users (name, username, email, user_address, member_code, user_birth_date, user_age_group_id, user_status_id, user_account_status_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	qryUpdate = `UPDATE st_users SET name = $2, username = $3, email = $4, user_address = $5, member_code = $6, user_birth_date = $7, user_age_group_id = $8, user_status_id = $9, user_account_status_id = $10, WHERE id = $1`
	qryDelete = `DELETE FROM st_users WHERE id = $1`
)

//id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, user_status_id, user_password, salt, verifier, user_account_status_id, created, modified
//id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, user_status_id, user_account_status_id, created, modified

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	handlerStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.User{}, qryGetAll, nil)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.User{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.User
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Name, record.Username, record.Email)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.User
	id := handlerStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.Name, record.Username, record.Email, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
