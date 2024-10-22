package handlerBooking

import (
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

const debugTag = "handlerBooking."

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
		log.Printf("%v.GetAll()2 %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	parentID, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	records := []models.Booking{}
	err = h.db.Select(&records, `SELECT atb.*, ebs.status, atbpcount.participants
	FROM public.at_bookings atb
	JOIN public.et_booking_status ebs on ebs.id=atb.booking_status_id
	LEFT JOIN (SELECT atbp.booking_id, COUNT(atbp.id) as participants
		FROM public.at_booking_people atbp
		GROUP BY atbp.booking_id) atbpcount ON atbpcount.booking_id=atb.id
	WHERE atb.trip_id = $1`, parentID)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetList()2 %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record := models.Booking{}
	err = h.db.Get(&record, `SELECT id, owner_id, trip_id, notes, from_date, to_date, booking_status_id, created, modified 
		FROM at_bookings WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get()2 %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Booking
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Beginx() // Start transaction
	if err != nil {
		http.Error(w, debugTag+"Create()1: Could not start transaction", http.StatusInternalServerError)
		return
	}

	err = tx.QueryRow(`
        INSERT INTO at_bookings (owner_id, trip_id, notes, from_date, to_date, booking_status_id) 
        VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		record.OwnerID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID,
	).Scan(&record.ID)

	if err != nil {
		tx.Rollback() // Rollback on error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.Commit() // Commit on success
	if err != nil {
		http.Error(w, debugTag+"Create()3: Could not commit transaction", http.StatusInternalServerError)
		return
	}

	//w.WriteHeader(http.StatusCreated)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec("DELETE FROM at_bookings WHERE id = $1", id)
	if err != nil {
		http.Error(w, debugTag+"Delete() - Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
