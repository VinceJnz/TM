package booking

import (
	"api-server/v2/localHandlers/handler"
	"api-server/v2/models"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

// BookingHandler handles requests for the Booking resource
type BookingHandler struct {
	*handler.BaseHandler[models.Booking] //???
}

// NewBookingHandler creates a new BookingHandler
func NewBookingHandler(db *sqlx.DB) *BookingHandler {
	baseHandler := handler.NewBaseHandler[models.Booking](db, "bookings", "id, owner_id, notes, from_date, to_date, booking_status_id, created, modified")
	return &BookingHandler{BaseHandler: baseHandler}
}

func (h *BookingHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	h.BaseHandler.GetAll(w, r, func(rows *sql.Rows) (models.Booking, error) {
		var booking models.Booking
		err := rows.Scan(&booking.ID, &booking.OwnerID, &booking.Notes, &booking.FromDate, &booking.ToDate, &booking.BookingStatusID, &booking.Created, &booking.Modified)
		return booking, err
	})
}

func (h *BookingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	h.BaseHandler.Get(w, r, id, func(row *sql.Row) (models.Booking, error) {
		var booking models.Booking
		err := row.Scan(&booking.ID, &booking.OwnerID, &booking.Notes, &booking.FromDate, &booking.ToDate, &booking.BookingStatusID, &booking.Created, &booking.Modified)
		return booking, err
	})
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var booking models.Booking
	h.BaseHandler.Create(w, r, func(booking *models.Booking) (int, error) {
		var id int
		err := h.DB.QueryRow(
			"INSERT INTO bookings (owner_id, notes, from_date, to_date, booking_status_id, created, modified) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
			booking.OwnerID, booking.Notes, booking.FromDate, booking.ToDate, booking.BookingStatusID, booking.Created, booking.Modified,
		).Scan(&id)
		return id, err
	}, &booking)
}

func (h *BookingHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	var booking models.Booking
	h.BaseHandler.Update(w, r, id, func(booking *models.Booking) error {
		booking.Modified = time.Now().UTC()
		_, err := h.DB.Exec(
			"UPDATE bookings SET owner_id = $1, notes = $2, from_date = $3, to_date = $4, booking_status_id = $5, modified = $6 WHERE id = $7",
			booking.OwnerID, booking.Notes, booking.FromDate, booking.ToDate, booking.BookingStatusID, booking.Modified, booking.ID,
		)
		return err
	}, &booking)
}

func (h *BookingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	h.BaseHandler.Delete(w, r, id)
}
