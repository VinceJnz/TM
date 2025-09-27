package handlerTrip

import (
	"encoding/json"
	"log"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerTrip."

const (
	qryGetAll = `SELECT att.*, ettd.level as difficulty_level, etts.status as trip_status, sum(atbcount.participants) as participants
					FROM public.at_trips att
					LEFT JOIN public.et_trip_difficulty ettd ON ettd.id=att.difficulty_level_id
					JOIN public.et_trip_status etts ON etts.id=att.trip_status_id
					LEFT JOIN (SELECT atb.*, COUNT(atb.id) as participants
						FROM public.at_bookings atb
						JOIN public.at_booking_people atbp ON atbp.booking_id=atb.id
						GROUP BY atb.id) atbcount ON atbcount.trip_id=att.id
					--WHERE att.owner_id = $1 OR true=$2
					GROUP BY att.id, ettd.level, etts.status`
	qryGet = `SELECT att.*, etts.status as trip_status
					FROM public.at_trips att
					LEFT JOIN public.et_trip_status etts ON etts.id=att.trip_status_id
					WHERE att.id = $1`
	qryCreate = `INSERT INTO at_trips (owner_id, trip_name, location, difficulty_level_id, from_date, to_date, max_participants, trip_status_id, trip_type_id, trip_cost_group_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	qryUpdate = `UPDATE at_trips SET (trip_name, location, difficulty_level_id, from_date, to_date, max_participants, trip_status_id, trip_type_id, trip_cost_group_id) = ($4, $5, $6, $7, $8, $9, $10, $11, $12) WHERE id = $1 AND (owner_id = $2 OR true=$3)`
	qryDelete = `DELETE FROM at_trips WHERE id = $1 AND (owner_id = $2 OR true=$3)`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	//session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session) // Used to retrieve the userID from the context so that access level can be assessed.
	//if !ok {
	//	log.Printf(debugTag+"GetAll()1 UserID not available in request context. userID=%v\n", session.UserID)
	//	http.Error(w, "UserID not available in request context", http.StatusInternalServerError)
	//	return
	//}
	//log.Printf(debugTag+"GetAll()2 session=%v\n", session)

	// Includes code to check if the user has access.
	//dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.Trip{}, qryGetAll, session.UserID, session.AdminFlag)
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.Trip{}, qryGetAll)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.Trip{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Trip
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, session.UserID, record.Name, record.Location, record.DifficultyID, record.FromDate, record.ToDate, record.MaxParticipants, record.TripStatusID, record.TripTypeID, record.TripCostGroupID)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.Trip
	id := dbStandardTemplate.GetID(w, r)
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	// This validation needs to be added into the process ???????????????????????????????????????
	if err := h.RecordValidation(record); err != nil {
		log.Printf(debugTag+"Update()2 validation error: %v", err)
		//http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": debugTag + "Update: " + err.Error()})
		return
	}

	log.Printf(debugTag + "Update()3 processing the update")
	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, id, session.UserID, session.AdminFlag, record.Name, record.Location, record.DifficultyID, record.FromDate, record.ToDate, record.MaxParticipants, record.TripStatusID, record.TripTypeID, record.TripCostGroupID)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id, session.UserID, session.AdminFlag)
}

func (h *Handler) RecordValidation(record models.Trip) error {
	return helpers.ValidateDatesFromLtTo(record.FromDate, record.ToDate)
}
