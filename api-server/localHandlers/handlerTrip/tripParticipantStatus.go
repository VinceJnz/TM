package handlerTrip

import (
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"net/http"
)

const (
	sqlGetParticipantStatus = `WITH booking_order AS (
			SELECT 
				atb.trip_id,
				atb.payment_date,
				atb.id as booking_id,
				atbp.id as participant_id,
				atbp.person_id as person_id,
				ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atb.payment_date ASC, atbp.id ASC) AS booking_position
			FROM at_booking_people atbp
			JOIN at_bookings atb ON atb.id=atbp.booking_id
			)
			SELECT 
				att.id AS trip_id,
				att.trip_name,
				att.from_date,
				att.to_date,
				att.max_participants,
				booking_order.booking_id,
				CASE WHEN stu.user_account_hidden IS NOT TRUE THEN participant_id END AS participant_id,
				CASE WHEN stu.user_account_hidden IS NOT TRUE THEN booking_order.person_id END AS person_id,
				CASE
				WHEN stu.user_account_hidden THEN 'name hidden'
                    ELSE stu.name
                END as person_name,
				CASE WHEN stu.user_account_hidden IS NOT TRUE THEN booking_position END AS booking_position,
				CASE
					WHEN stu.user_account_hidden THEN NULL
					WHEN (booking_position <= att.max_participants AND payment_date IS NOT NULL)  THEN 'before_threshold_paid' 
					WHEN (booking_position <= att.max_participants)  THEN 'before_threshold' 
					WHEN (booking_position > att.max_participants) THEN 'after_threshold'
					ELSE 'n/a'
				END AS booking_status
			FROM at_trips att
			LEFT JOIN booking_order ON att.id=booking_order.trip_id
			LEFT JOIN st_users stu ON stu.id=booking_order.person_id
			ORDER BY trip_id, booking_order.booking_position;`
)

// GetBookingStatus: retrieves and returns all records with the status of each users booking (trip participant booking status list)
func (h *Handler) GetParticipantStatus(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	dbStandardTemplate.GetList(w, r, debugTag, h.appConf.Db, &[]models.TripParticipantStatus{}, sqlGetParticipantStatus)
}
