package handlerMyBookings

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerMyBookings."

const (
	qryGetAll = `SELECT atb.*,
				ebs.status, COUNT(stu.name) as participants,
				SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
				--SUM(attc.amount) AS booking_cost,
				att.trip_name, stu2.name as owner_name
				FROM at_trips att
				LEFT JOIN at_bookings atb ON atb.trip_id=att.id
				LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
					 JOIN public.et_booking_status ebs on ebs.id=atb.booking_status_id
				LEFT JOIN st_users stu ON stu.id=atbp.person_id
				LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
				LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
										AND attc.member_status_id=stu.member_status_id
										AND attc.user_age_group_id=stu.user_age_group_id
				LEFT JOIN st_users stu2 ON stu2.id=atb.owner_id
				WHERE atb.owner_id = $1 --OR true=$2 -- TODO: Can this be removed? Should users only see their own bookings in this endpoint?
				GROUP BY att.id, att.trip_name, atb.id, stu2.name, ebs.status
				ORDER BY att.trip_name, atb.id`

	qryGet = `SELECT id, owner_id, trip_id, notes, from_date, to_date, booking_status_id, ebs.status, ab.booking_date, ab.payment_date, ab.booking_price, created, modified 
					FROM at_bookings WHERE id = $1`

	qryGetList = `SELECT atb.*,
					ebs.status, COUNT(stu.name) as participants, SUM(attc.amount) AS booking_cost, att.trip_name
				FROM at_trips att
				LEFT JOIN at_bookings atb ON atb.trip_id=att.id
				LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
					 JOIN public.et_booking_status ebs on ebs.id=atb.booking_status_id
				LEFT JOIN st_users stu ON stu.id=atbp.person_id
				LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
				LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
										AND attc.member_status_id=stu.member_status_id
										AND attc.user_age_group_id=stu.user_age_group_id
				WHERE atb.trip_id = $1
				GROUP BY att.id, att.trip_name, atb.id, ebs.status
				ORDER BY att.trip_name, atb.id`

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
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)

	// Includes code to check if the user has access. ???????? Query needs to be checked ???????????????????
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.MyBooking{}, qryGetAll, session.UserID)
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

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, session.UserID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID)

	// Send notification email to user
	h.appConf.EmailSvc.SendMail(session.Email, "New Booking Created", fmt.Sprintf("A new booking has been created by user ID %d for trip %d, %s.", session.UserID, record.TripID.Int64, record.TripName))
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.Booking

	id := dbStandardTemplate.GetID(w, r)
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Update()1 dest=%+v", record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	record.ID = id

	//Need to add this validation in ?????????????????????????????????
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
		return fmt.Errorf(debugTag+"ParentRecordValidation()1 - Record not found: error message = %s, parentID = %v", err.Error(), parentID)
	} else if err != nil {
		return fmt.Errorf(debugTag+"ParentRecordValidation()2 - Internal Server Error:  error message = %s, parentID = %v", err.Error(), parentID)
	}

	if record.FromDate.Before(validationRecord.FromDate) {
		return fmt.Errorf("dateError: Booking From-date is before Trip From-date")
	}

	if record.ToDate.After(validationRecord.ToDate) {
		return fmt.Errorf("dateError: Booking To-date is after Trip To-date")
	}

	return nil
}
