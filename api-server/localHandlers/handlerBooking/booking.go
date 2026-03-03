package handlerBooking

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerBooking."

const (
	qryGetAll = `SELECT stu2.username AS owner_username, atb.*, ebs.status, COUNT(stu.name) as participants,
			SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
			--SUM(attc.amount) AS booking_cost,
			att.trip_name
		FROM at_trips att
			LEFT JOIN at_bookings atb ON atb.trip_id=att.id
			LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
				JOIN public.et_booking_status ebs on ebs.id=atb.booking_status_id
			LEFT JOIN st_users stu ON stu.id=atbp.person_id
			LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
			LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
									AND attc.member_status_id=stu.member_status_id
									AND attc.user_age_group_id=stu.user_age_group_id
			JOIN st_users stu2 ON stu2.id=atb.owner_id
		WHERE atb.owner_id = $1 OR $2 IN ('admin', 'sysadmin')
		GROUP BY stu2.username, att.id, att.trip_name, atb.id, ebs.status
		ORDER BY stu2.username, att.trip_name, atb.id;`

	qryGet = `SELECT id, owner_id, trip_id, notes, from_date, to_date, booking_status_id, ebs.status, ab.booking_date, ab.payment_date, ab.booking_price, created, modified 
					FROM at_bookings WHERE id = $1`

	qryGetList = `SELECT stu2.username AS owner_username, atb.*,
			ebs.status, COUNT(stu.name) as participants,
			--SUM(attc.amount) AS booking_cost,
			SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
			att.trip_name
		FROM at_trips att
			LEFT JOIN at_bookings atb ON atb.trip_id=att.id
			LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
				JOIN public.et_booking_status ebs on ebs.id=atb.booking_status_id
			LEFT JOIN st_users stu ON stu.id=atbp.person_id
			LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
			LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
									AND attc.member_status_id=stu.member_status_id
									AND attc.user_age_group_id=stu.user_age_group_id
			JOIN st_users stu2 ON stu2.id=atb.owner_id
		WHERE atb.trip_id = $1
		GROUP BY stu2.username, att.id, att.trip_name, atb.id, ebs.status
		ORDER BY stu2.username, att.trip_name, atb.id`

	qryCreate = `INSERT INTO at_bookings (owner_id, trip_id, notes, from_date, to_date, booking_status_id) 
        			VALUES ($1, $2, $3, $4, $5, $6) 
					RETURNING id`
	qryUpdateAdmin = `UPDATE at_bookings 
					SET (owner_id, trip_id, notes, from_date, to_date, booking_status_id, booking_date, payment_date, booking_price) = ($2, $3, $4, $5, $6, $7, $8, $9, $10)
					WHERE id = $1`
	qryUpdate = `UPDATE at_bookings 
					SET (notes, from_date, to_date, booking_status_id) = ($4, $5, $6, $7)
					WHERE id = $1 AND (owner_id = $2 OR $3 IN ('admin', 'sysadmin'))`
	qryDelete = `DELETE FROM at_bookings WHERE id = $1 AND (owner_id = $2 OR $3 IN ('admin', 'sysadmin'))`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseUrlBooking, baseUrlTrip string) {
	helpers.AddRouteGroup(r, baseUrlBooking, h)
	h.RegisterRoutesStripe(r, baseUrlBooking)
	r.HandleFunc(baseUrlTrip+"/{id:[0-9]+}"+baseUrlBooking, h.GetList).Methods("GET")
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Returns owner bookings, or all bookings for admin/sysadmin roles.
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGetAll, session.UserID, session.Role)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGet, id)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGetList, id)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Booking
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		log.Printf(debugTag+"Create err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Create: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, session.UserID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.Booking

	id := dbStandardTemplate.GetID(w, r)
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		log.Printf(debugTag+"Update err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if session.Role == "admin" || session.Role == "sysadmin" {
		dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdateAdmin, id, record.OwnerID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID, record.BookingDate, record.PaymentDate, record.BookingPrice)
	} else {
		dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, id, session.UserID, session.Role, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID)
	}
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := dbStandardTemplate.GetID(w, r)
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id, session.UserID, session.Role)
}

func (h *Handler) RecordValidation(record models.Booking) error {
	if err := helpers.ValidateDatesFromLtTo(record.FromDate, record.ToDate); err != nil {
		return err
	}
	if err := h.ParentRecordValidation(record); err != nil {
		return err
	}
	return nil
}

const (
	sqlBookingParentRecordValidation = `SELECT * FROM at_trips WHERE id = $1`
)

func (h *Handler) ParentRecordValidation(record models.Booking) error {
	parentID := record.TripID

	validationRecord := models.Trip{}
	err := h.appConf.Db.Get(&validationRecord, sqlBookingParentRecordValidation, parentID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("trip not found for trip_id=%v", parentID)
	} else if err != nil {
		log.Printf("%sParentRecordValidation() database error for trip_id=%v: %v", debugTag, parentID, err)
		return fmt.Errorf("unable to validate trip record")
	}

	if record.FromDate.Before(validationRecord.FromDate) {
		return fmt.Errorf("dateError: Booking From-date is before Trip From-date")
	}

	if record.ToDate.After(validationRecord.ToDate) {
		return fmt.Errorf("dateError: Booking To-date is after Trip To-date")
	}

	return nil
}
