package handlerUserPayments

import (
	"api-server/v2/app"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerUserPayments."

type Handler struct {
	appConf *app.Config
}

func New(appConf *app.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.UserPayments{}
	err := h.appConf.Db.Select(&records, `SELECT atup.*
	FROM public.at_user_payments atup`)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf(debugTag+"GetAll()2 %v\n", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	bookingID, err := helpers.GetIDFromRequest(r)
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	records := []models.UserPayments{}
	err = h.appConf.Db.Select(&records, `SELECT atup.*
	FROM public.at_user_payments atup
	WHERE atup.booking_id = $1`, bookingID)
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

	record := models.UserPayments{}
	err = h.appConf.Db.Get(&record, `SELECT id, user_id, booking_id, payment_date, amount, payment_method, created, modified 
		FROM at_user_payments WHERE id = $1`, id)
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
	var record models.UserPayments
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tx, err := h.appConf.Db.Beginx() // Start transaction
	if err != nil {
		http.Error(w, debugTag+"Create()1: Could not start transaction", http.StatusInternalServerError)
		return
	}

	err = tx.QueryRow(`
        INSERT INTO at_user_payments (user_id, booking_id, payment_date, amount, payment_method) 
        VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		record.UserID, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod,
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

	var record models.UserPayments
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	_, err = h.appConf.Db.Exec(`
		UPDATE at_user_payments
		SET user_id = $1, booking_id = $2, payment_date = $3, amount = $4, payment_method = $5
		WHERE id = $6`,
		record.UserID, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod,
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

	_, err = h.appConf.Db.Exec("DELETE FROM at_user_payments WHERE id = $1", id)
	if err != nil {
		http.Error(w, debugTag+"Delete() - Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
