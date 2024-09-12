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

const debug = "handlerBooking"

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

// GetAll retrieves all bookings
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	bookings := []models.Booking{}
	err := h.db.Select(bookings, `SELECT id, owner_id, notes, from_date, to_date, booking_status_id, created, modified	FROM bookings`)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetAll()2 %v\n", debug, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(bookings)
}

// Get retrieves a single booking by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debug, err)
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	booking := models.Booking{}
	err = h.db.Get(&booking, `SELECT id, owner_id, notes, from_date, to_date, booking_status_id, created, modified 
		FROM bookings WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get()2 %v\n", debug, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(booking)
}

// Create adds a new booking
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var booking models.Booking
	if err := json.NewDecoder(r.Body).Decode(&booking); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	//now := time.Now().UTC()
	//booking.Created = now
	//booking.Modified = now

	err := h.db.QueryRow(`
		INSERT INTO bookings (owner_id, notes, from_date, to_date, booking_status_id) 
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		booking.OwnerID, booking.Notes, booking.FromDate, booking.ToDate, booking.BookingStatusID,
	).Scan(&booking.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}

// Update modifies an existing booking
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	var booking models.Booking
	if err := json.NewDecoder(r.Body).Decode(&booking); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	booking.ID = id
	//booking.Modified = time.Now().UTC()

	_, err = h.db.Exec(`
		UPDATE bookings 
		SET owner_id = $1, notes = $2, from_date = $3, to_date = $4, booking_status_id = $5 
		WHERE id = $6`,
		booking.OwnerID, booking.Notes, booking.FromDate, booking.ToDate, booking.BookingStatusID,
		booking.ID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(booking)
}

// Delete removes a booking
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec("DELETE FROM bookings WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
