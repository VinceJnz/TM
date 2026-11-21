package handlerUser

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/profiles"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	store "api-server/v2/modelMethods/profiles"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"

	"github.com/gorilla/mux"
)

const debugTag = "handlerUser."

const (
	qryGetAll = `SELECT id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, member_status_id, user_account_status_id, user_account_hidden, created, modified FROM st_users`
	qryGet    = `SELECT id, name, username, email, user_address, member_code, user_birth_date, user_age_group_id, member_status_id, user_account_status_id, user_account_hidden, created, modified FROM st_users WHERE id = $1`
	qryCreate = `INSERT INTO st_users (name, username, email, user_address, member_code, user_birth_date, user_age_group_id, member_status_id, user_account_status_id, user_account_hidden) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
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
	helpers.AddRouteGroup(r, baseURL, h, "users")
	helpers.CapabilityRoute(r, baseURL+"/set-username", "users", h.SetUsername).Methods("POST")
	helpers.CapabilityRoute(r, baseURL+"/lookup", "users", h.Lookup).Methods("GET")
}

func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID <= 0 {
		helpers.WriteError(w, 401, "unauthorized", "Authentication required")
		return
	}
	if session.AccessScope != "any" {
		helpers.WriteError(w, 403, "forbidden", "User lookup requires any scope")
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	id := 0
	if raw := r.URL.Query().Get("id"); raw != "" {
		var err error
		id, err = strconv.Atoi(raw)
		if err != nil || id < 1 {
			helpers.WriteError(w, 400, "invalid_id", "Invalid user ID")
			return
		}
	}
	if len(query) > 100 {
		helpers.WriteError(w, 400, "invalid_search", "Search must be at most 100 bytes")
		return
	}
	items, err := resourcequery.UserOptions(r.Context(), h.appConf.Db, query, id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, items)
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.User{}, qryGetAll)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.User{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.User
	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		log.Printf(debugTag+"Create err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(record.MemberCode.String) == "" {
		memberCode, err := helpers.GenerateMemberCode(h.appConf.Db)
		if err != nil {
			log.Printf("%sCreate failed to generate member code: %v", debugTag, err)
			http.Error(w, "failed to generate member code", http.StatusInternalServerError)
			return
		}
		record.MemberCode.SetValid(memberCode)
	}

	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Name, record.Username, record.Email, record.Address, record.MemberCode, record.BirthDate, record.UserAgeGroupID, record.MemberStatusID, record.AccountStatusID, record.AccountHidden)
	if record.ID > 0 {
		if err := helpers.NotifyAdminsUserReviewRequired(
			debugTag+"Create:",
			h.appConf.Db,
			h.appConf.EmailService(),
			h.appConf.Settings.AdminNotifyGroup,
			record.ID,
			record.Username,
			record.Email.String,
			record.Name,
		); err != nil {
			log.Printf("%vCreate failed to send admin notification: %v", debugTag, err)
		}
	}
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.User
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		log.Printf(debugTag+"Update err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, id, record.Name, record.Username, record.Email, record.Address, record.MemberCode, record.BirthDate, record.UserAgeGroupID, record.MemberStatusID, record.AccountStatusID, record.AccountHidden)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	childChecks := []dbStandardTemplate.ChildCheckQuery{
		{
			TableName: "bookings (owner)",
			Query:     "SELECT COUNT(*) FROM at_bookings WHERE owner_id = $1",
		},
		{
			TableName: "booking participants",
			Query:     "SELECT COUNT(*) FROM at_booking_people WHERE person_id = $1",
		},
		{
			TableName: "payments",
			Query:     "SELECT COUNT(*) FROM at_payments p JOIN at_bookings b ON b.id = p.booking_id WHERE b.owner_id = $1",
		},
		{
			TableName: "security group memberships",
			Query:     "SELECT COUNT(*) FROM st_user_group WHERE user_id = $1",
		},
		{
			TableName: "trips (owner)",
			Query:     "SELECT COUNT(*) FROM at_trips WHERE owner_id = $1",
		},
		{
			TableName: "group bookings (owner)",
			Query:     "SELECT COUNT(*) FROM at_group_bookings WHERE owner_id = $1",
		},
	}

	dbStandardTemplate.DeleteWithChildCheck(w, r, debugTag, h.appConf.Db, qryDelete, childChecks, id)
}

// SetUsername allows the authenticated user to set their username (unique).
func (h *Handler) SetUsername(w http.ResponseWriter, r *http.Request) {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}
	var input struct {
		Username string `json:"username"`
	}
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid username request")
		return
	}
	username, err := (service.Service{Store: store.Store{DB: h.appConf.Db}}).SetUsername(r.Context(), session.UserID, input.Username)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			helpers.WriteError(w, 400, "invalid_username", "Username must contain 3 to 20 bytes")
		} else {
			helpers.WriteDataError(w, err)
		}
		return
	}
	helpers.WriteJSON(w, 200, map[string]string{"status": "ok", "username": username})
}
