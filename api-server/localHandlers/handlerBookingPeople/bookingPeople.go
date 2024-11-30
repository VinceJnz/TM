package handlerBookingPeople

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"

	"github.com/gorilla/mux"
)

const debugTag = "handlerBookingPeople."

const (
	qryGetAll = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
					FROM at_booking_people bp
						JOIN st_users p ON p.id=bp.person_id`
	qryGet = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
				FROM at_booking_people bp
					JOIN st_users p ON p.id=bp.person_id
				WHERE bp.id = $1`
	qryCreate = `INSERT INTO at_booking_people (owner_id, booking_id, person_id, notes) VALUES ($1, $2, $3, $4) RETURNING id`
	qryUpdate = `UPDATE at_booking_people
					SET owner_id = $1, booking_id = $2, person_id = $3, notes = $4
					WHERE id = $5`
	qryDelete = `DELETE FROM at_booking_people WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	handlerStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.BookingPeople{}, qryGetAll, nil)

	records := []models.BookingPeople{}
	err := h.appConf.Db.Select(&records, `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
	FROM at_booking_people bp
		JOIN st_users p ON p.id=bp.person_id`)
	if err == sql.ErrNoRows {
		log.Printf("%v.GetAll()1 %v\n", debugTag, err)
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf(debugTag+"GetAll()2 %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	parentID, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.GetList()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	records := []models.BookingPeople{}
	err = h.appConf.Db.Select(&records, `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
	FROM at_booking_people bp
		JOIN st_users p ON p.id=bp.person_id
	WHERE bp.booking_id = $1`, parentID)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetList()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.BookingPeople{}, qryGet, id)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.BookingPeople
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.OwnerID, record.BookingID, record.PersonID, record.Notes)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.BookingPeople
	id := handlerStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.OwnerID, record.BookingID, record.PersonID, record.Notes, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
