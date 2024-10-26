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
	err := h.db.Select(&records, `SELECT attc.id, attc.trip_cost_group_id, attc.description, attc.user_status_id, etus.status as user_status, attc.user_category_id, etuc.category as user_category, attc.season_id, ets.season, attc.amount, attc.created, attc.modified
	FROM public.at_trip_costs attc
	LEFT JOIN et_user_status etus on etus.id = attc.user_status_id
	JOIN et_user_categorys etuc on etuc.id = attc.user_category_id
	JOIN et_seasons ets on ets.id = attc.season_id`)

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
	err = h.db.Get(&record, `SELECT *
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
        INSERT INTO at_trip_costs (trip_cost_group_id, user_status_id, user_category_id, season_id, amount) 
        VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		record.TripCostGroupID, record.UserStatusID, record.UserCategoryID, record.SeasonID, record.Amount,
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
        SET trip_cost_group_id = $1, user_status_id = $2, user_category_id = $3, season_id = $4, amount = $5
        WHERE id = $6`,
		record.TripCostGroupID, record.UserStatusID, record.UserCategoryID, record.SeasonID, record.Amount, record.ID,
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
