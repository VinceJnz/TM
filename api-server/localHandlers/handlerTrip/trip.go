package handlerTrip

import (
	"encoding/json"
	"log"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
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
					WHERE att.owner_id = $1 OR true=$2
					GROUP BY att.id, ettd.level, etts.status`
	qryGet = `SELECT att.*, etts.status as trip_status
					FROM public.at_trips att
					LEFT JOIN public.et_trip_status etts ON etts.id=att.trip_status_id
					WHERE att.id = $1`
	qryCreate = `INSERT INTO at_trips (trip_name, location, from_date, to_date, max_participants, trip_status_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	qryUpdate = `UPDATE at_trips SET trip_name = $1, location = $2, from_date = $3, to_date = $4, max_participants = $5, trip_status_id = $6 WHERE id = $7`
	qryDelete = `DELETE FROM at_trips WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(models.Session) // Used to retrieve the userID from the context so that access level can be assessed.
	if !ok {
		log.Printf(debugTag+"GetAll()1 UserID not available in request context. userID=%v\n", session.UserID)
		http.Error(w, "UserID not available in request context", http.StatusInternalServerError)
		return
	}
	log.Printf(debugTag+"GetAll()2 session=%v\n", session)

	// Includes code to check if the user has access.
	handlerStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.Trip{}, qryGetAll, session.UserID, session.AdminFlag)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.Trip{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Trip
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Name, record.Location, record.FromDate, record.ToDate, record.MaxParticipants, record.TripStatusID)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.Trip
	id := handlerStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	// This validation needs to be added into the process ???????????????????????????????????????
	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.Name, record.Location, record.FromDate, record.ToDate, record.MaxParticipants, record.TripStatusID, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}

func (h *Handler) RecordValidation(record models.Trip) error {
	return helpers.ValidateDatesFromLtTo(record.FromDate, record.ToDate)
}
