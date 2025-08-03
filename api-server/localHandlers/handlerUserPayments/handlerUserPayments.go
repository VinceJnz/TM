package handlerUserPayments

import (
	"api-server/v2/app/appCore"
	"api-server/v2/dbTemplates/dbStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
)

const debugTag = "handlerUserPayments."

const (
	qryGetAll  = `SELECT * FROM at_user_payments`
	qryGet     = `SELECT * FROM at_user_payments WHERE id = $1`
	qryGetList = `SELECT atup.*
					FROM public.at_user_payments atup
					WHERE atup.booking_id = $1`
	qryCreate = `INSERT INTO at_user_payments (user_id, booking_id, payment_date, amount, payment_method) 
        			VALUES ($1, $2, $3, $4, $5)
					RETURNING id`
	qryUpdate = `UPDATE at_user_payments
					SET user_id = $1, booking_id = $2, payment_date = $3, amount = $4, payment_method = $5
					WHERE id = $6`
	qryDelete = `DELETE FROM at_user_payments WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	dbStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.UserPayments{}, qryGetAll)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.UserPayments{}, qryGetList, id)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.UserPayments{}, qryGet, id)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.UserPayments
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.UserID, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.UserPayments
	id := dbStandardTemplate.GetID(w, r)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.UserID, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
