package handlerGroupBooking

import (
	"api-server/v2/app"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerGroupBooking."

type Handler struct {
	appConf *app.Config
}

func New(appConf *app.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all group bookings
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.GroupBooking{}
	err := h.appConf.Db.Select(&records, `SELECT id, group_name, owner_id, created, modified FROM at_group_bookings`)
	if err == sql.ErrNoRows {
		http.Error(w, "No group bookings found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetAll() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a single group booking identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record := models.GroupBooking{}
	err = h.appConf.Db.Get(&record, `SELECT id, group_name, owner_id, created, modified 
                             FROM at_group_bookings WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "Group booking not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get() failed to execute query: %v\n", debugTag, err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Create: adds a new group booking
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.GroupBooking
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := h.appConf.Db.Beginx()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	err = tx.QueryRow(`
		INSERT INTO at_group_bookings (group_name, owner_id) 
		VALUES ($1, $2) RETURNING id`,
		record.GroupName, record.OwnerID,
	).Scan(&record.ID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// Update: modifies an existing group booking
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var record models.GroupBooking
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	// Begin transaction
	tx, err := h.appConf.Db.Beginx()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
		UPDATE at_group_bookings 
		SET group_name = $1, owner_id = $2 
		WHERE id = $3`,
		record.GroupName, record.OwnerID, record.ID,
	)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Delete: removes a group booking identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := h.appConf.Db.Beginx()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM at_group_bookings WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"result": "success"})
}
