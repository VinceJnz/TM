package handlerTripCostGroup

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerTripCostGroup."

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves all trip costs
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.TripCostGroup{}
	err := h.appConf.Db.Select(&records, `SELECT *
                                  FROM at_trip_cost_groups`)
	if err == sql.ErrNoRows {
		http.Error(w, "No trip costs found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%vGetAll() failed to execute query: %v\n", debugTag, err)
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
		log.Printf("%vGet() failed to retrieve ID: %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record := models.TripCost{}
	err = h.appConf.Db.Get(&record, `SELECT *
                             FROM at_trip_cost_groups WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "Trip cost not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%vGet() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Create: adds a new trip cost record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.TripCostGroup
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tx, err := h.appConf.Db.Beginx()
	if err != nil {
		http.Error(w, "Could not start transaction"+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.QueryRow(`
        INSERT INTO at_trip_cost_groups (description) 
        VALUES ($1) RETURNING id`,
		record.Description,
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

	var record models.TripCostGroup
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	_, err = h.appConf.Db.Exec(`
        UPDATE at_trip_cost_groups 
        SET description = $1
        WHERE id = $1`,
		record.Description, record.ID,
	)
	if err != nil {
		log.Printf("%vUpdate() failed to execute query: %v\n", debugTag, err)
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

	_, err = h.appConf.Db.Exec("DELETE FROM at_trip_cost_groups WHERE id = $1", id)
	if err != nil {
		log.Printf("%vDelete() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
