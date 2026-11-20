package handlerBookingPeople

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"

	"github.com/gorilla/mux"
)

const debugTag = "handlerBookingPeople."

const (
	qryGetAll = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
					FROM at_booking_people bp
						JOIN st_users p ON p.id=bp.person_id JOIN at_bookings b ON b.id=bp.booking_id WHERE (b.owner_id=$1 OR LOWER($2)='any')`
	qryGet = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
				FROM at_booking_people bp
					JOIN st_users p ON p.id=bp.person_id JOIN at_bookings b ON b.id=bp.booking_id
				WHERE bp.id = $1 AND (b.owner_id=$2 OR LOWER($3)='any')`
	qryGetList = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
				FROM at_booking_people bp
					JOIN st_users p ON p.id=bp.person_id JOIN at_bookings b ON b.id=bp.booking_id
				WHERE bp.booking_id = $1 AND (b.owner_id=$2 OR LOWER($3)='any')`
	qryCreate = `INSERT INTO at_booking_people (owner_id, booking_id, person_id, notes) SELECT $1,b.id,$3,$4 FROM at_bookings b WHERE b.id=$2 AND (b.owner_id=$1 OR LOWER($5)='any') AND b.booking_status_id=1 AND b.stripe_session_id IS NULL FOR UPDATE OF b RETURNING id`
	qryUpdate = `UPDATE at_booking_people
					SET (person_id, notes) = ($4, $5)
					WHERE id = $1 AND EXISTS (SELECT 1 FROM at_bookings b WHERE b.id=at_booking_people.booking_id AND (b.owner_id=$2 OR LOWER($3)='any') AND b.booking_status_id=1 AND b.stripe_session_id IS NULL FOR UPDATE OF b)`
	qryDelete = `DELETE FROM at_booking_people WHERE id = $1 AND EXISTS (SELECT 1 FROM at_bookings b WHERE b.id=at_booking_people.booking_id AND (b.owner_id=$2 OR LOWER($3)='any') AND b.booking_status_id=1 AND b.stripe_session_id IS NULL FOR UPDATE OF b)`
)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h, "bookingPeople")
	helpers.CapabilityRoute(r, "/bookings/{id:[0-9]+}/bookingPeople", "bookingPeople", h.GetList).Methods("GET")
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID <= 0 {
		return
	}
	records := []models.BookingPeople{}
	err := h.appConf.Db.Select(&records, qryGetAll, session.UserID, session.AccessScope)
	if err == sql.ErrNoRows {
		log.Printf("%v.GetAll %v\n", debugTag, err)
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf(debugTag+"GetAll %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(records); err != nil {
		log.Printf("%sGetAll failed to encode response: %v", debugTag, err)
	}
}

// Get: retrieves and returns a list of records identified by parent id
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID <= 0 {
		return
	}
	params := mux.Vars(r)
	parentID, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.GetList %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	records := []models.BookingPeople{}
	err = h.appConf.Db.Select(&records, qryGetList, parentID, session.UserID, session.AccessScope)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetList %v\n", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(records); err != nil {
		log.Printf("%sGetList failed to encode response: %v", debugTag, err)
	}
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID <= 0 {
		return
	}
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.BookingPeople{}, qryGet, id, session.UserID, session.AccessScope)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.BookingPeople
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

	if err := h.RecordValidation(session, record); err != nil {
		http.Error(w, debugTag+"Create: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	dbStandardTemplate.Create(w, r, debugTag, h.appConf.Db, &record.ID, qryCreate, session.UserID, record.BookingID, record.PersonID, record.Notes, session.AccessScope)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var record models.BookingPeople
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := helpers.DecodeJSONBody(r, &record); err != nil {
		log.Printf(debugTag+"Update decode error: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	record.ID = id

	if err := h.RecordValidation(session, record); err != nil {
		http.Error(w, debugTag+"Update: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	dbStandardTemplate.Update(w, r, debugTag, h.appConf.Db, &record, qryUpdate, id, session.UserID, session.AccessScope, record.PersonID, record.Notes)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var err error
	var record models.BookingPeople
	id := dbStandardTemplate.GetID(w, r)
	if id <= 0 {
		return
	}
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validation stuff
	record.ID = id
	err = h.appConf.Db.Get(&record, qryGet, record.ID, session.UserID, session.AccessScope)
	if err != nil {
		http.Error(w, debugTag+"Delete - Record not found: ", http.StatusUnprocessableEntity)
		return
	}

	if err := h.RecordValidation(session, record); err != nil {
		http.Error(w, debugTag+"Delete: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	dbStandardTemplate.Delete(w, r, debugTag, h.appConf.Db, nil, qryDelete, id, session.UserID, session.AccessScope)
}

const (
	qryParentRecordValidation = `SELECT id,owner_id,booking_status_id,stripe_session_id FROM at_bookings WHERE id = $1`
)

func (h *Handler) RecordValidation(session *models.Session, record models.BookingPeople) error {
	var err error

	parentID := record.BookingID
	validationRecord := models.Booking{}
	err = h.appConf.Db.Get(&validationRecord, qryParentRecordValidation, parentID)
	switch {
	case err == sql.ErrNoRows:
		return fmt.Errorf("booking not found for booking_id=%v", parentID)
	case err != nil:
		log.Printf("%sParentRecordValidation() database error for booking_id=%v: %v", debugTag, parentID, err)
		return fmt.Errorf("unable to validate booking record")
	case validationRecord.BookingStatusID != int(models.Not_paid) || validationRecord.StripeSessionID.Valid:
		return fmt.Errorf("booking is locked for payment")
	case validationRecord.OwnerID != session.UserID && !dbStandardTemplate.CanAccessAny(session):
		return fmt.Errorf("access denied")
	default:
		return nil
	}
}
