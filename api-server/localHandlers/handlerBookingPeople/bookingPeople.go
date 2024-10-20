package handlerBookingPeople

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

const debugTag = "handlerBookingPeople."

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.BookingPeople{}
	err := h.db.Select(&records, `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
	FROM at_booking_people bp
		JOIN st_users p ON p.id=bp.person_id`)
	if err == sql.ErrNoRows {
		log.Printf("%v.GetAll()1 %v\n", debugTag, err)
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetAll()2 %v\n", debugTag, err)
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
	err = h.db.Select(&records, `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
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
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	record := models.BookingPeople{}
	err = h.db.Get(&record, `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
	FROM at_booking_people bp
		JOIN st_users p ON p.id=bp.person_id
	WHERE bp.id = $1`, id)
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
	var record models.BookingPeople
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf("%v.Create()1 %v\n", debugTag, err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(`
		INSERT INTO at_booking_people (owner_id, booking_id, person_id, notes) 
		VALUES ($1, $2, $3, $4) RETURNING id`,
		record.OwnerID, record.BookingID, record.PersonID, record.Notes,
	).Scan(&record.ID)
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
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	var record models.BookingPeople
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf("%v.Update()1 %v\n", debugTag, err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	_, err = h.db.Exec(`
		UPDATE at_booking_people
		SET owner_id = $1, booking_id = $2, person_id = $3, notes = $4
		WHERE id = $5`,
		record.OwnerID, record.BookingID, record.PersonID, record.Notes,
		record.ID,
	)
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
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec("DELETE FROM at_booking_people WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
