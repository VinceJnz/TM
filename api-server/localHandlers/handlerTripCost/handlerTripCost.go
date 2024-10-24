package handlerTripCost

import (
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

const debugTag = "handlerTripCost."

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

// GetAll: retrieves all trip costs
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.TripCost{}
	err := h.db.Select(&records, `SELECT id, trip_id, user_age_group_id, season_id, amount, created, modified
                                  FROM at_trip_costs`)

	if err == sql.ErrNoRows {
		http.Error(w, "No trip costs found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetAll() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves a single trip cost by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get() failed to retrieve ID: %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record := models.TripCost{}
	err = h.db.Get(&record, `SELECT id, trip_id, user_age_group_id, season_id, amount, created, modified
                             FROM at_trip_costs WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "Trip cost not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Create: adds a new trip cost record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.TripCost
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		http.Error(w, "Could not start transaction"+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.QueryRow(`
        INSERT INTO at_trip_costs (trip_id, user_age_group_id, season_id, amount) 
        VALUES ($1, $2, $3, $4) RETURNING id`,
		record.TripID, record.UserAgeGroupID, record.SeasonID, record.Amount,
	).Scan(&record.ID)

	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Could not commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Update: modifies the existing trip cost by ID
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid trip cost ID", http.StatusBadRequest)
		return
	}

	var record models.TripCost
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	_, err = h.db.Exec(`
        UPDATE at_trip_costs 
        SET trip_id = $1, user_age_group_id = $2, season_id = $3, amount = $4
        WHERE id = $5`,
		record.TripID, record.UserAgeGroupID, record.SeasonID, record.Amount, record.ID,
	)
	if err != nil {
		log.Printf("%v.Update() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Delete: removes a trip cost by ID
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid trip cost ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec("DELETE FROM at_trip_costs WHERE id = $1", id)
	if err != nil {
		log.Printf("%v.Delete() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
