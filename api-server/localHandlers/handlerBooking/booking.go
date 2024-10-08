package handlerBooking

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

const debug = "handlerBooking."

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.Booking{}
	err := h.db.Select(&records, `SELECT ab.id, ab.owner_id, ab.trip_id, ab.notes, ab.from_date, ab.to_date, ab.booking_status_id, ebs.status, ab.created, ab.modified
	FROM public.at_bookings ab
	JOIN public.et_booking_status ebs on ebs.id=ab.booking_status_id`)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetAll()2 %v\n", debug, err)
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
		log.Printf("%v.Get()1 %v\n", debug, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	record := models.Booking{}
	err = h.db.Get(&record, `SELECT id, owner_id, trip_id, notes, from_date, to_date, booking_status_id, created, modified 
		FROM at_bookings WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get()2 %v\n", debug, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Booking
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(`
		INSERT INTO at_bookings (owner_id, trip_id, notes, from_date, to_date, booking_status_id) 
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		record.OwnerID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID,
	).Scan(&record.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	var record models.Booking
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	_, err = h.db.Exec(`
		UPDATE at_bookings 
		SET owner_id = $1, trip_id = $2, notes = $3, from_date = $4, to_date = $5, booking_status_id = $6 
		WHERE id = $7`,
		record.OwnerID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID,
		record.ID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec("DELETE FROM at_bookings WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
