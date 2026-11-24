package paymentLedger

import (
	"api-server/v2/modelMethods/trips"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrParticipantDetails = errors.New("complete required participant details before checkout")

func requireParticipantDetails(ctx context.Context, tx *sqlx.Tx, booking int) error {
	var row struct {
		Required bool `db:"requires_participant_details"`
		TripID   int  `db:"trip_id"`
	}
	if err := tx.GetContext(ctx, &row, `SELECT requires_participant_details, trip_id FROM at_bookings WHERE id=$1`, booking); err != nil {
		return err
	}
	if !row.Required {
		return nil
	}
	questionnaire, err := trips.Questionnaire(ctx, tx, row.TripID, 0)
	if err != nil {
		return err
	}
	var participants []struct {
		models.ParticipantDetails
		Raw []byte `db:"trip_answers"`
	}
	err = tx.SelectContext(ctx, &participants, `
		SELECT COALESCE(submitted_name,'') AS submitted_name,
			COALESCE(mobile_phone,'') AS mobile_phone, COALESCE(home_address,'') AS home_address,
			COALESCE(membership_category,'') AS membership_category,
			COALESCE(emergency_contact_name,'') AS emergency_contact_name,
			COALESCE(emergency_contact_phone,'') AS emergency_contact_phone,
			COALESCE(dietary_requirements,'') AS dietary_requirements, trip_answers
		FROM at_booking_people WHERE booking_id=$1`, booking)
	if err != nil {
		return err
	}
	if len(participants) == 0 {
		return ErrParticipantDetails
	}
	for _, participant := range participants {
		var answers map[string]json.RawMessage
		if err := json.Unmarshal(participant.Raw, &answers); err != nil {
			return err
		}
		if !participant.Valid() || questionnaire.NeedsAnswers(answers) {
			return ErrParticipantDetails
		}
	}
	return nil
}
