package handlerTrip

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"api-server/v2/models"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

const debugTag = "handlerTrip."

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.Trip{}
	//err := h.db.Select(&records, `SELECT att.*, etts.status as trip_status
	//FROM public.at_trips att
	//LEFT JOIN public.et_trip_status etts ON etts.id=att.trip_status_id`)

	err := h.db.Select(&records, `SELECT att.*, etts.status as trip_status, sum(atbcount.participants) as participants
	FROM public.at_trips att
	JOIN public.et_trip_status etts ON etts.id=att.trip_status_id
	LEFT JOIN (SELECT atb.*, COUNT(atb.id) as participants
		FROM public.at_bookings atb
		JOIN public.at_booking_people atbp ON atbp.booking_id=atb.id
		GROUP BY atb.id) atbcount ON atbcount.trip_id=att.id
	GROUP BY att.id, etts.status`)

	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetAll()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	record := models.Trip{}
	err = h.db.Get(&record, `SELECT att.*, etts.status as trip_status
	FROM public.at_trips att
	LEFT JOIN public.et_trip_status etts ON etts.id=att.trip_status_id
	WHERE att.id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Trip
	json.NewDecoder(r.Body).Decode(&record)

	err := h.db.QueryRow(
		"INSERT INTO at_trips (trip_name, location, from_date, to_date, max_participants, trip_status_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		record.Name, record.Location, record.FromDate, record.ToDate, record.MaxParticipants, record.TripStatusID).Scan(&record.ID)
	if err != nil {
		log.Printf("%v.Create()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Update()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	var record models.Trip
	json.NewDecoder(r.Body).Decode(&record)
	record.ID = id

	_, err = h.db.Exec("UPDATE at_trips SET trip_name = $1, location = $2, from_date = $3, to_date = $4, max_participants = $5, trip_status_id = $6 WHERE id = $7",
		record.Name, record.Location, record.FromDate, record.ToDate, record.MaxParticipants, record.TripStatusID, record.ID)
	if err != nil {
		log.Printf("%v.Update()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Delete()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec("DELETE FROM at_trips WHERE id = $1", id)
	if err != nil {
		log.Printf("%v.Delete()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
