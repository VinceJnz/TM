package handlerSeasons

import (
	"api-server/v2/app/appCore"
	"api-server/v2/dbTemplates/dbStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerSeasons."

const (
	qryGetAll = `SELECT * FROM et_seasons`
	qryGet    = `SELECT * FROM et_seasons WHERE id = $1`
	qryCreate = `INSERT INTO et_seasons (season)
					VALUES ($1)
					RETURNING id`
	qryUpdate = `UPDATE et_seasons 
					SET season = $1 
					WHERE id = $2`
	qryDelete = `DELETE FROM et_seasons WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.Season{}, qryGetAll)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.Season{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Season
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Season)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.Season
	id := dbStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.Season, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
