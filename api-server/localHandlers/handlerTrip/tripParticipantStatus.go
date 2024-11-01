package handlerTrip

import (
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// GetBookingStatus: retrieves and returns all records with the status of each users booking (trip participant booking status list)
func (h *Handler) GetParticipantStatus(w http.ResponseWriter, r *http.Request) {
	records := []models.TripParticipantStatus{}

	err := h.appConf.Db.Select(&records, `WITH booking_order AS (
    SELECT 
        atb.trip_id,
		atb.id as booking_id,
        atbp.id as participant_id,
        atbp.person_id as person_id,
	    ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atbp.id ASC) AS booking_position
    FROM public.at_booking_people atbp
	JOIN public.at_bookings atb ON atb.id=atbp.booking_id
	)
	SELECT 
		trip_id,
		att.trip_name,
		att.from_date,
		att.to_date,
		booking_order.booking_id,
		participant_id,
		booking_order.person_id,
		stu.name as person_name,
		CASE 
			WHEN booking_position <= att.max_participants THEN 'before_threshold' 
			ELSE 'after_threshold' 
		END AS booking_status
	FROM booking_order
	JOIN public.at_trips att ON att.id=booking_order.trip_id
	JOIN public.st_users stu ON stu.id=booking_order.person_id
	ORDER BY trip_id, booking_position;`)

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
