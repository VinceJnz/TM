package handlerGroupBooking

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerGroupBooking."

const (
	qryGetAll = `SELECT id, group_name, owner_id, created, modified FROM at_group_bookings`
	qryGet    = `SELECT id, group_name, owner_id, created, modified FROM at_group_bookings WHERE id = $1`
	qryCreate = `INSERT INTO at_group_bookings (group_name, owner_id) VALUES ($1, $2) RETURNING id`
	qryUpdate = `UPDATE at_group_bookings SET group_name = $1, owner_id = $2 WHERE id = $3`
	qryDelete = `DELETE FROM at_group_bookings WHERE id = $1`
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
}

// GetAll: retrieves and returns all group bookings
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.GroupBooking{}, qryGetAll)
}

// Get: retrieves and returns a single group booking identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.GroupBooking{}, qryGet, id)
}

// Create: adds a new group booking
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.GroupBooking
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.GroupName, record.OwnerID)
}

// Update: modifies an existing group booking
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.GroupBooking
	id := dbStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.GroupName, record.OwnerID, id)
}

// Delete: removes a group booking identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
