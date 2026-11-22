// Package tripreport owns the participant report query and its stable row order.
package tripreport

import (
	"api-server/v2/models"
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"strings"
)

type Store struct{ DB *sqlx.DB }

const participantQuery = `WITH booking_order AS (
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

// Report rows have no resource ID. The query orders by trip and the unique
// booking position, including an empty row for trips without participants.
func (s Store) List(ctx context.Context, limit, offset int) ([]models.TripParticipantStatus, bool, error) {
	return s.list(ctx, limit, offset, 0, false)
}

func (s Store) Trips(ctx context.Context, limit, offset int) ([]models.TripParticipantStatus, bool, error) {
	return s.list(ctx, limit, offset, 0, true)
}
func (s Store) Participants(ctx context.Context, id, limit, offset int) ([]models.TripParticipantStatus, bool, error) {
	if id <= 0 {
		return nil, false, errors.New("invalid trip")
	}
	return s.list(ctx, limit, offset, id, false)
}

func (s Store) list(ctx context.Context, limit, offset, id int, summaries bool) ([]models.TripParticipantStatus, bool, error) {
	if limit < 1 || limit > 500 || offset < 0 || offset > 1000000 {
		return nil, false, errors.New("invalid report page")
	}
	items := []models.TripParticipantStatus{}
	query := strings.TrimSuffix(strings.TrimSpace(participantQuery), ";")
	args := []any{limit + 1, offset}
	if summaries {
		query = `SELECT id AS trip_id,trip_name,from_date,to_date,max_participants FROM at_trips ORDER BY id`
	}
	if id > 0 {
		query = strings.Replace(query, "ORDER BY trip_id,", "WHERE att.id=$3 ORDER BY trip_id,", 1)
		args = append(args, id)
	}
	query += " LIMIT $1 OFFSET $2"
	if err := s.DB.SelectContext(ctx, &items, query, args...); err != nil {
		return nil, false, err
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	return items, more, nil
}
