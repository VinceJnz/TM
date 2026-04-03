package handlerUserPayments

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerUserPayments."

const (
	qryGetAll  = `SELECT * FROM at_payments`
	qryGet     = `SELECT * FROM at_payments WHERE id = $1`
	qryGetList = `SELECT atup.*
					FROM public.at_payments atup
					WHERE atup.booking_id = $1`
	qryCreate = `INSERT INTO at_payments (booking_id, payment_date, amount, payment_method) 
					VALUES ($1, $2, $3, $4)
					RETURNING id`
	qryUpdate = `UPDATE at_payments
					SET booking_id = $1, payment_date = $2, amount = $3, payment_method = $4
					WHERE id = $5`
	qryDelete = `DELETE FROM at_payments WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.UserPayments]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.UserPayments]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.UserPayments{} },
		NewRecord:   func() *models.UserPayments { return &models.UserPayments{} },
		IDDest:      func(record *models.UserPayments) any { return &record.ID },
		SetID:       func(record *models.UserPayments, id int) { record.ID = id },
		CreateArgs: func(record *models.UserPayments) []any {
			return []any{record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod}
		},
		UpdateArgs: func(id int, record *models.UserPayments) []any {
			return []any{record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod, id}
		},
	})
	return h
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
	r.HandleFunc(baseURL+"/booking/{id:[0-9]+}", h.GetList).Methods("GET")
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	h.crud.GetAll(w, r)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.UserPayments{}, qryGetList, id)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.crud.Get(w, r)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.UserPayments
	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		log.Printf(debugTag+"Create err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tx, err := h.appConf.Db.Beginx()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tx.QueryRow(qryCreate, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod).Scan(&record.ID); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := helpers.SyncBookingPaymentStatusTx(tx, record.BookingID); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, record.ID)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.UserPayments
	id := dbStandardTemplate.GetID(w, r)
	if id == 0 {
		return
	}
	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		log.Printf(debugTag+"Update err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	tx, err := h.appConf.Db.Beginx()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	oldBookingID := 0
	if err := tx.Get(&oldBookingID, `SELECT booking_id FROM at_payments WHERE id = $1`, id); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			http.Error(w, "Record not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(qryUpdate, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod, id); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := helpers.SyncBookingPaymentStatusTx(tx, record.BookingID); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if oldBookingID != record.BookingID {
		if err := helpers.SyncBookingPaymentStatusTx(tx, oldBookingID); err != nil {
			tx.Rollback()
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, record)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	if id == 0 {
		return
	}

	tx, err := h.appConf.Db.Beginx()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	bookingID := 0
	if err := tx.Get(&bookingID, `SELECT booking_id FROM at_payments WHERE id = $1`, id); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			http.Error(w, "Record not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if _, err := tx.Exec(qryDelete, id); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := helpers.SyncBookingPaymentStatusTx(tx, bookingID); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
