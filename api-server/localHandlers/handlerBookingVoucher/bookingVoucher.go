package handlerBookingVoucher

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const debugTag = "handlerBookingVoucher."

const (
	qryGetAllActive = `SELECT id, code, discount_percent, fixed_cost, expiry_date, is_active, created, modified
					FROM at_booking_vouchers
					WHERE is_active = true
					AND (expiry_date IS NULL OR expiry_date >= CURRENT_DATE)
					ORDER BY code`

	qryGetAllAdmin = `SELECT id, code, discount_percent, fixed_cost, expiry_date, is_active, created, modified
					FROM at_booking_vouchers
					ORDER BY code`

	qryGet = `SELECT id, code, discount_percent, fixed_cost, expiry_date, is_active, created, modified
				FROM at_booking_vouchers
				WHERE id = $1`

	qryCreate = `INSERT INTO at_booking_vouchers (code, discount_percent, fixed_cost, expiry_date, is_active)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id`

	qryUpdate = `UPDATE at_booking_vouchers
				SET code = $1, discount_percent = $2, fixed_cost = $3, expiry_date = $4, is_active = $5
				WHERE id = $6`

	qryDelete = `DELETE FROM at_booking_vouchers WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

func (h *Handler) RegisterRoutesProtected(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL, h.GetAll).Methods("GET")
}

func (h *Handler) RegisterRoutesAdmin(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/{id:[0-9]+}", h.Get).Methods("GET")
	r.HandleFunc(baseURL, h.Create).Methods("POST")
	r.HandleFunc(baseURL+"/{id:[0-9]+}", h.Update).Methods("PUT")
	r.HandleFunc(baseURL+"/{id:[0-9]+}", h.Delete).Methods("DELETE")
}

func normalizeVoucher(v *models.BookingVoucher) {
	v.Code = strings.ToUpper(strings.TrimSpace(v.Code))
	if v.DiscountPercent != nil {
		discountPercent := *v.DiscountPercent
		if discountPercent < 0 {
			discountPercent = 0
		}
		if discountPercent > 100 {
			discountPercent = 100
		}
		v.DiscountPercent = &discountPercent
	}
	if v.FixedCost != nil {
		fixedCost := *v.FixedCost
		if fixedCost < 0 {
			fixedCost = 0
		}
		v.FixedCost = &fixedCost
	}
}

func validateVoucher(v *models.BookingVoucher) error {
	hasDiscount := v.DiscountPercent != nil
	hasFixed := v.FixedCost != nil

	if hasDiscount == hasFixed {
		return fmt.Errorf("exactly one of discount_percent or fixed_cost is required")
	}

	return nil
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session != nil && (session.Role == "admin" || session.Role == "sysadmin") {
		dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.BookingVoucher{}, qryGetAllAdmin)
		return
	}
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.BookingVoucher{}, qryGetAllActive)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.BookingVoucher{}, qryGet, id)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.BookingVoucher
	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	normalizeVoucher(&record)
	if record.Code == "" {
		http.Error(w, "Voucher code is required", http.StatusUnprocessableEntity)
		return
	}
	if err := validateVoucher(&record); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.Code, record.DiscountPercent, record.FixedCost, record.ExpiryDate, record.IsActive)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	var record models.BookingVoucher
	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	normalizeVoucher(&record)
	if record.Code == "" {
		http.Error(w, "Voucher code is required", http.StatusUnprocessableEntity)
		return
	}
	if err := validateVoucher(&record); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	record.ID = id

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.Code, record.DiscountPercent, record.FixedCost, record.ExpiryDate, record.IsActive, id)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
}
