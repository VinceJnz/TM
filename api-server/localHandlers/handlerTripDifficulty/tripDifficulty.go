package handlerTripDifficulty

import (
	"api-server/v2/app/appCore"
	"api-server/v2/dbTemplates/handlerStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerTripDifficulty."

const (
	qryGetAll = `SELECT * FROM et_trip_difficulty`
	qryGet    = `SELECT * FROM et_trip_difficulty WHERE id = $1`
	qryCreate = `INSERT INTO et_trip_difficulty (level, level_short, description)
					VALUES ($1, $2, $3) 
					RETURNING id`
	qryUpdate = `UPDATE et_trip_difficulty 
					SET level = $1, level_short = $2, description = $3
					WHERE id = $4`
	qryDelete = `DELETE FROM et_trip_difficulty WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	handlerStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.TripDificulty{}, qryGetAll)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.TripDificulty{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.TripDificulty
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Level, record.LevelShort, record.Description)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.TripDificulty
	id := handlerStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.Level, record.LevelShort, record.Description, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
