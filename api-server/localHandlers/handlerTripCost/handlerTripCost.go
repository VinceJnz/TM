package handlerTripCost

import (
	"api-server/v2/app/appCore"
	"api-server/v2/dbTemplates/dbStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerTripCost."

const (
	qryGetAll = `SELECT attc.id, attc.trip_cost_group_id, attc.description, attc.member_status_id, etms.status as member_status, attc.user_age_group_id, etuag.age_group as user_age_group, attc.season_id, ets.season, attc.amount, attc.created, attc.modified
					FROM public.at_trip_costs attc
					LEFT JOIN et_member_status etms on etms.id = attc.member_status_id
					JOIN et_user_age_groups etuag on etuag.id = attc.user_age_group_id
					JOIN et_seasons ets on ets.id = attc.season_id`
	qryGet    = `SELECT * FROM at_trip_costs WHERE id = $1`
	qryCreate = `INSERT INTO at_trip_costs (trip_cost_group_id, member_status_id, user_age_group_id, season_id, amount) 
			        VALUES ($1, $2, $3, $4, $5) 
					RETURNING id`
	qryUpdate = `UPDATE at_trip_costs 
					SET trip_cost_group_id = $1, member_status_id = $2, user_age_group_id = $3, season_id = $4, amount = $5
					WHERE id = $6`
	qryDelete = `DELETE FROM at_trip_costs WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves all trip costs
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.TripCost{}, qryGetAll)
}

// Get: retrieves a single trip cost by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.TripCost{}, qryGet, id)
}

// Create: adds a new trip cost record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.TripCost
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.TripCostGroupID, record.MemberStatusID, record.UserAgeGroupID, record.SeasonID, record.Amount)
}

// Update: modifies the existing trip cost by ID
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.TripCost
	id := dbStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.TripCostGroupID, record.MemberStatusID, record.UserAgeGroupID, record.SeasonID, record.Amount, id)
}

// Delete: removes a trip cost by ID
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
