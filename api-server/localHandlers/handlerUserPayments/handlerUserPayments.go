package handlerUserPayments

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"net/http"

	"github.com/gorilla/mux"
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
			return []any{record.UserID, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod}
		},
		UpdateArgs: func(id int, record *models.UserPayments) []any {
			return []any{record.UserID, record.BookingID, record.PaymentDate, record.Amount, record.PaymentMethod, id}
		},
	})
	return h
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
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
	h.crud.Create(w, r)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	h.crud.Update(w, r)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	h.crud.Delete(w, r)
}
