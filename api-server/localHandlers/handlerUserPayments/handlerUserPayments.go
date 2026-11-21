package handlerUserPayments

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/modelMethods/paymentLedger"
	"api-server/v2/models"

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
	helpers.AddRouteGroup(r, baseURL, h, "payments")
	helpers.CapabilityRoute(r, baseURL+"/booking/{id:[0-9]+}", "payments", h.GetList).Methods("GET")
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	h.crud.GetAll(w, r)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.UserPayments{}, qryGetList, id)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	h.crud.Get(w, r)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.write(w, r, "create") }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.write(w, r, "update") }
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) { h.write(w, r, "delete") }
func (h *Handler) write(w http.ResponseWriter, r *http.Request, operation string) {
	if !helpers.RequireFinancialAccess(w, r, h.appConf.SessionIDKey) {
		return
	}
	id := 0
	if operation != "create" {
		id = dbStandardTemplate.GetID(w, r)
		if id <= 0 {
			return
		}
	}
	var input paymentLedger.PaymentWrite
	if operation != "delete" {
		if err := helpers.DecodeJSONBody(r, &input); err != nil {
			helpers.WriteError(w, 400, "invalid_request", "Invalid request payload")
			return
		}
	}
	id, err := paymentLedger.WritePayment(r.Context(), h.appConf.Db, operation, id, input)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	if operation == "delete" {
		w.WriteHeader(200)
		return
	}
	if operation == "create" {
		helpers.WriteJSON(w, 201, id)
		return
	}
	response := models.UserPayments{ID: id, BookingID: input.BookingID, PaymentDate: input.PaymentDate, Amount: input.Amount, PaymentMethod: input.PaymentMethod}
	helpers.WriteJSON(w, 200, response)
}
