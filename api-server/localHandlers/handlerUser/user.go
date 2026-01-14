package handlerUser

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"

	"github.com/gorilla/mux"
)

const debugTag = "handlerUser."

const (
	qryGetAll = `SELECT id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, member_status_id, user_account_status_id, user_account_hidden, created, modified FROM st_users`
	qryGet    = `SELECT id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, member_status_id, user_account_status_id, user_account_hidden, created, modified FROM st_users WHERE id = $1`
	qryCreate = `INSERT INTO st_users (name, username, email, user_address, member_code, user_birth_date, user_age_group_id, member_status_id, user_account_status_id, user_account_hidden) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	qryUpdate = `UPDATE st_users SET (name, username, email, user_address, member_code, user_birth_date, user_age_group_id, member_status_id, user_account_status_id, user_account_hidden) = ($2, $3, $4, $5, $6, $7, $8, $9, $10, $11) WHERE id = $1`
	qryDelete = `DELETE FROM st_users WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
	r.HandleFunc(baseURL+"/set-username", h.SetUsername).Methods("POST")
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.User{}, qryGetAll)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.User{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.User
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Name, record.Username, record.Email, record.Address, record.MemberCode, record.BirthDate, record.UserAgeGroupID, record.MemberStatusID, record.AccountStatusID, record.AccountHidden)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.User
	id := dbStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, id, record.Name, record.Username, record.Email, record.Address, record.MemberCode, record.BirthDate, record.UserAgeGroupID, record.MemberStatusID, record.AccountStatusID, record.AccountHidden)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}

// SetUsername allows the authenticated user to set their username (unique).
func (h *Handler) SetUsername(w http.ResponseWriter, r *http.Request) {
	// Require a session (set by RequireOAuthOrSessionAuth middleware)
	sessI := r.Context().Value(h.appConf.SessionIDKey)
	if sessI == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sess, ok := sessI.(*models.Session)
	if !ok {
		http.Error(w, "invalid session", http.StatusInternalServerError)
		return
	}
	// Parse request body
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	uname := strings.TrimSpace(req.Username)
	if uname == "" || len(uname) < 3 || len(uname) > 20 {
		http.Error(w, "invalid username", http.StatusBadRequest)
		return
	}
	// Check uniqueness
	existing, err := dbAuthTemplate.UserNameReadQry(debugTag+"SetUsername:check ", h.appConf.Db, uname)
	if err == nil {
		if existing.ID != sess.UserID {
			http.Error(w, "username taken", http.StatusConflict)
			return
		}
		// If it is the same user, continue (no-op)
	}
	// Load user and update
	user, err := dbAuthTemplate.UserReadQry(debugTag+"SetUsername ", h.appConf.Db, sess.UserID)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	user.Username = uname
	_, err = dbAuthTemplate.UserWriteQry(debugTag+"SetUsername ", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v failed to set username: %v", debugTag, err)
		http.Error(w, "failed to update username", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "username": uname})
}
