package handlerWebAuthnManagement

import (
	"encoding/json"
	"log"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerWebAuthnManagement."

//const (
//	qryGetAll = `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata, created, modified FROM st_webauthn_credentials`
//	qryGet    = `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata, created, modified FROM st_webauthn_credentials WHERE id = $1`
//	qryCreate = `INSERT INTO st_webauthn_credentials (user_id, credential_id, credential_data, last_used, device_name, device_metadata) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
//	qryUpdate = `UPDATE st_webauthn_credentials SET (user_id, credential_id, credential_data, last_used, device_name, device_metadata) = ($2, $3, $4, $5, $6, $7) WHERE id = $1`
//	qryDelete = `DELETE FROM st_webauthn_credentials WHERE id = $1`
//)

const (
	qryGetAll = `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata, created, modified FROM st_webauthn_credentials
	WHERE (true=$1 OR user_id = $2)`
	qryGet = `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata, created, modified FROM st_webauthn_credentials
	WHERE id = $1 AND (true=$2 OR user_id = $3)`
	qryCreate = `INSERT INTO st_webauthn_credentials (user_id, credential_id, credential_data, last_used, device_name, device_metadata) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	qryUpdate = `UPDATE st_webauthn_credentials SET (user_id, credential_id, credential_data, last_used, device_name, device_metadata) = ($3, $4, $5, $6, $7, $8) WHERE id = $1
	AND (true=$2 OR user_id = $3)`
	qryDelete = `DELETE FROM st_webauthn_credentials WHERE id = $1 AND (true=$2 OR user_id = $3)`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.WebAuthnCredential{}, qryGetAll, session.AdminFlag, session.UserID)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.WebAuthnCredential{}, qryGet, id, session.AdminFlag, session.UserID)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.WebAuthnCredential
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	record.UserID = session.UserID
	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.UserID, record.CredentialID, record.Credential, record.LastUsed, record.DeviceName, record.DeviceMetadata)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.WebAuthnCredential
	id := dbStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	record.UserID = session.UserID
	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, id, session.AdminFlag, record.UserID, record.CredentialID, record.Credential, record.LastUsed, record.DeviceName, record.DeviceMetadata)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id, session.AdminFlag, session.UserID)
}
