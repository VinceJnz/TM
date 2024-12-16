package handlerBooking

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const debugTag = "handlerBooking."

const (
	qryGetAll = `SELECT ab.id, ab.owner_id, ab.trip_id, ab.notes, ab.from_date, ab.to_date, ab.booking_status_id, ebs.status, ab.booking_date, ab.payment_date, ab.booking_price, ab.created, ab.modified
					FROM public.at_bookings ab
						JOIN public.et_booking_status ebs on ebs.id=ab.booking_status_id
					WHERE ab.owner_id = $1 OR true=$2`
	qryGet = `SELECT id, owner_id, trip_id, notes, from_date, to_date, booking_status_id, ebs.status, ab.booking_date, ab.payment_date, ab.booking_price, created, modified 
					FROM at_bookings WHERE id = $1`
	qryGetList = `SELECT atb.*, ebs.status, atbpcount.participants
					FROM public.at_bookings atb
					JOIN public.et_booking_status ebs on ebs.id=atb.booking_status_id
					LEFT JOIN (SELECT atbp.booking_id, COUNT(atbp.id) as participants
						FROM public.at_booking_people atbp
						GROUP BY atbp.booking_id) atbpcount ON atbpcount.booking_id=atb.id
					WHERE atb.trip_id = $1`
	qryCreate = `INSERT INTO at_bookings (owner_id, trip_id, notes, from_date, to_date, booking_status_id) 
        			VALUES ($1, $2, $3, $4, $5, $6) 
					RETURNING id`
	qryUpdate = `UPDATE at_bookings 
					SET owner_id = $1, trip_id = $2, notes = $3, from_date = $4, to_date = $5, booking_status_id = $6 
					WHERE id = $7`
	qryDelete = `DELETE FROM at_bookings WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(models.Session)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}
	log.Printf(debugTag+"GetAll()1 userID=%v, adminFlag=%v\n", session.UserID, session.AdminFlag)

	// Includes code to check if the user has access. ???????? Query needs to be checked ???????????????????
	//handlerStandardTemplate.GetAll(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGetAll, session.UserID, session.AdminFlag)
	handlerStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGetAll, session.UserID, session.AdminFlag)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Get(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGet, id)
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.Booking{}, qryGetList, id)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.Booking
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		log.Printf(debugTag+"Create()2 err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.RecordValidation(record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	//log.Printf(debugTag+"Create()3 record = %+v, query = %+v", record, qryCreate)

	handlerStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, record.OwnerID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.Booking
	id := handlerStandardTemplate.GetID(w, r)

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

	handlerStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, record.OwnerID, record.TripID, record.Notes, record.FromDate, record.ToDate, record.BookingStatusID, id)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r)
	handlerStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id)
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
	//sqlBookingParentRecordValidation = `SELECT id, owner_id, trip_id, notes, from_date, to_date, booking_status_id, created, modified FROM at_trips WHERE id = $1`
	//sqlBookingParentRecordValidation = `SELECT id, owner_id, notes, from_date, to_date, booking_status_id, created, modified FROM at_trips WHERE id = $1`
	sqlBookingParentRecordValidation = `SELECT * FROM at_trips WHERE id = $1`
)

func (h *Handler) ParentRecordValidation(record models.Booking) error {
	parentID := record.TripID

	validationRecord := models.Trip{}
	err := h.appConf.Db.Get(&validationRecord, sqlBookingParentRecordValidation, parentID)
	if err == sql.ErrNoRows {
		return fmt.Errorf(debugTag+"ParentRecordValidation()1 - Record not found: error message = %s", err.Error())
	} else if err != nil {
		return fmt.Errorf(debugTag+"ParentRecordValidation()2 - Internal Server Error:  error message = %s", err.Error())
	}

	if record.FromDate.Before(validationRecord.FromDate) {
		return fmt.Errorf("dateError: Booking From-date is before Trip From-date")
	}

	if record.ToDate.After(validationRecord.ToDate) {
		return fmt.Errorf("dateError: Booking To-date is after Trip To-date")
	}

	return nil
}
