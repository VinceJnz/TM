package handlerTripCostGroup

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerTripCostGroup."

const (
	qryGetAll = `SELECT * FROM at_trip_cost_groups`
	qryGet    = `SELECT * FROM at_trip_cost_groups WHERE id = $1`
	qryCreate = `INSERT INTO at_trip_cost_groups (description)
					VALUES ($1)
					RETURNING id`
	qryUpdate = `UPDATE at_trip_cost_groups 
					SET description = $1
					WHERE id = $2`
	qryDelete = `DELETE FROM at_trip_cost_groups WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves all trip costs
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	handlerStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.TripCostGroup{}, qryGetAll, nil)
}

// Get: retrieves a single trip cost by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.TripCostGroup{}, qryGet, id)
}

// Create: adds a new trip cost record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.TripCostGroup
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Description)
}

// Update: modifies the existing trip cost by ID
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.TripCostGroup
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.Description, id)
}

// Delete: removes a trip cost by ID
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
