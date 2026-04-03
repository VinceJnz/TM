package handlerPaymentRefunds

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

const debugTag = "handlerPaymentRefunds."

const (
	qryGetAll  = `SELECT * FROM at_refunds`
	qryGet     = `SELECT * FROM at_refunds WHERE id = $1`
	qryGetList = `SELECT apr.*
					FROM public.at_refunds apr
					WHERE apr.payment_id = $1`
	qryCreate = `INSERT INTO at_refunds (payment_id, refund_date, amount, reason, external_ref)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id`
	qryUpdate = `UPDATE at_refunds
				SET payment_id = $1, refund_date = $2, amount = $3, reason = $4, external_ref = $5
				WHERE id = $6`
	qryDelete = `DELETE FROM at_refunds WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.PaymentRefund]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.PaymentRefund]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.PaymentRefund{} },
		NewRecord:   func() *models.PaymentRefund { return &models.PaymentRefund{} },
		IDDest:      func(record *models.PaymentRefund) any { return &record.ID },
		SetID:       func(record *models.PaymentRefund, id int) { record.ID = id },
		CreateArgs: func(record *models.PaymentRefund) []any {
			return []any{record.PaymentID, record.RefundDate, record.Amount, record.Reason, record.ExternalRef}
		},
		UpdateArgs: func(id int, record *models.PaymentRefund) []any {
			return []any{record.PaymentID, record.RefundDate, record.Amount, record.Reason, record.ExternalRef, id}
		},
	})
	return h
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
	r.HandleFunc(baseURL+"/payment/{id:[0-9]+}", h.GetList).Methods("GET")
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	h.crud.GetAll(w, r)
}

func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.PaymentRefund{}, qryGetList, id)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.crud.Get(w, r)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.PaymentRefund
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

	if err := tx.QueryRow(qryCreate, record.PaymentID, record.RefundDate, record.Amount, record.Reason, record.ExternalRef).Scan(&record.ID); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	bookingID, err := helpers.GetBookingIDByPaymentIDTx(tx, record.PaymentID)
	if err != nil {
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

	helpers.WriteJSON(w, http.StatusCreated, record.ID)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.PaymentRefund
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

	oldBookingID, err := helpers.GetBookingIDByRefundIDTx(tx, id)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			http.Error(w, "Record not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(qryUpdate, record.PaymentID, record.RefundDate, record.Amount, record.Reason, record.ExternalRef, id); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	newBookingID, err := helpers.GetBookingIDByPaymentIDTx(tx, record.PaymentID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := helpers.SyncBookingPaymentStatusTx(tx, newBookingID); err != nil {
		tx.Rollback()
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if oldBookingID != newBookingID {
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

	bookingID, err := helpers.GetBookingIDByRefundIDTx(tx, id)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			http.Error(w, "Record not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
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
