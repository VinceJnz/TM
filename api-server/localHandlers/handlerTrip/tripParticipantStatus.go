package handlerTrip

import (
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

const (
	OLD2sqlGetParticipantStatus = `WITH booking_order AS (
			SELECT 
				atb.trip_id,
				atb.booking_status_id,
				atb.id as booking_id,
				atbp.id as participant_id,
				atbp.person_id as person_id,
				ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atb.booking_status_id DESC, atbp.id ASC) AS booking_position
			FROM public.at_booking_people atbp
			JOIN public.at_bookings atb ON atb.id=atbp.booking_id
			)
			SELECT 
				att.id AS trip_id,
				att.trip_name,
				att.from_date,
				att.to_date,
				att.max_participants,
				booking_order.booking_id,
				participant_id,
				booking_order.person_id,
				stu.name as person_name,
				--booking_order.booking_status_id,
				booking_position,
				CASE
					WHEN (booking_position <= att.max_participants AND booking_status_id = 3)  THEN 'before_threshold_paid' 
					WHEN (booking_position <= att.max_participants AND booking_status_id = 1)  THEN 'before_threshold' 
					WHEN (booking_position > att.max_participants) THEN 'after_threshold'
					ELSE 'n/a'
				END AS booking_status
			FROM public.at_trips att
			LEFT JOIN booking_order ON att.id=booking_order.trip_id
			LEFT JOIN public.st_users stu ON stu.id=booking_order.person_id
			ORDER BY trip_id, booking_position;`

	sqlGetParticipantStatus = `WITH booking_order AS (
			SELECT 
				atb.trip_id,
				atb.booking_status_id,
				atb.payment_date,
				atb.id as booking_id,
				atbp.id as participant_id,
				atbp.person_id as person_id,
				--ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atb.booking_date ASC, atb.booking_status_id DESC, atbp.id ASC) AS booking_position
				ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atb.payment_date ASC, atbp.id ASC) AS booking_position
			FROM public.at_booking_people atbp
			JOIN public.at_bookings atb ON atb.id=atbp.booking_id
			)
			SELECT 
				att.id AS trip_id,
				att.trip_name,
				att.from_date,
				att.to_date,
				att.max_participants,
				booking_order.booking_id,
				participant_id,
				booking_order.person_id,
				stu.name as person_name,
				--booking_order.booking_status_id,
				booking_position,
				CASE
					WHEN (booking_position <= att.max_participants AND payment_date IS NOT NULL)  THEN 'before_threshold_paid' 
					WHEN (booking_position <= att.max_participants)  THEN 'before_threshold' 
					WHEN (booking_position > att.max_participants) THEN 'after_threshold'
					ELSE 'n/a'
				END AS booking_status
			FROM public.at_trips att
			LEFT JOIN booking_order ON att.id=booking_order.trip_id
			LEFT JOIN public.st_users stu ON stu.id=booking_order.person_id
			ORDER BY trip_id, booking_position;`
)

// GetBookingStatus: retrieves and returns all records with the status of each users booking (trip participant booking status list)
func (h *Handler) GetParticipantStatus(w http.ResponseWriter, r *http.Request) {
	records := []models.TripParticipantStatus{}

	err := h.appConf.Db.Select(&records, sqlGetParticipantStatus)

	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.GetBookingStatus()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}
