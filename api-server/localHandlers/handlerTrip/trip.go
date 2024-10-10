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
	err := h.db.Select(&records, `SELECT * FROM at_trips`)
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
	err = h.db.Get(&record, "SELECT * FROM at_trips WHERE id = $1", id)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Trip
	json.NewDecoder(r.Body).Decode(&record)

	err := h.db.QueryRow(
		"INSERT INTO at_trips (trip_name, location, from_date, to_date, max_participants) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		record.Name, record.Location, record.FromDate, record.ToDate, record.MaxParticipants).Scan(&record.ID)
	if err != nil {
		log.Printf("%v.Create()2 %v\n", debugTag, err)
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
		log.Printf("%v.Update()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	var record models.Trip
	json.NewDecoder(r.Body).Decode(&record)
	record.ID = id

	_, err = h.db.Exec("UPDATE at_trips SET trip_name = $1, location = $2, from_date = $3, to_date = $4, max_participants = $5 WHERE id = $6",
		record.Name, record.Location, record.FromDate, record.ToDate, record.MaxParticipants, record.ID)
	if err != nil {
		log.Printf("%v.Update()2 %v\n", debugTag, err)
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
