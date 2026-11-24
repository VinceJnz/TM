package participants

import (
	"api-server/v2/modelMethods/trips"
	"api-server/v2/models"
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"
)

func (s Store) Questionnaire(ctx context.Context, booking, actor int, scope string) (models.Questionnaire, error) {
	var trip int
	err := s.DB.GetContext(ctx, &trip, `
		SELECT trip_id FROM at_bookings WHERE id=$1 AND (owner_id=$2 OR $3='any')`,
		booking, actor, scope)
	if err != nil {
		return models.Questionnaire{}, err
	}
	return trips.Questionnaire(ctx, s.DB, trip, 0)
}

func (s Store) Details(ctx context.Context, participant, actor int, scope string, leader bool) (models.ParticipantDetails, error) {
	var row struct {
		models.ParticipantDetails
		Answers   []byte `db:"trip_answers"`
		Questions []byte `db:"questions"`
		TripID    int    `db:"trip_id"`
	}
	err := s.DB.GetContext(ctx, &row, `
		SELECT COALESCE(bp.submitted_name, '') AS submitted_name,
			COALESCE(bp.mobile_phone, '') AS mobile_phone,
			COALESCE(bp.home_address, '') AS home_address,
			COALESCE(bp.membership_category, '') AS membership_category,
			COALESCE(bp.emergency_contact_name, '') AS emergency_contact_name,
			COALESCE(bp.emergency_contact_phone, '') AS emergency_contact_phone,
			COALESCE(bp.dietary_requirements, '') AS dietary_requirements,
			COALESCE(bp.notes, '') AS notes,
			COALESCE(bp.questionnaire_revision, 0) AS questionnaire_revision, bp.trip_answers,
			t.id AS trip_id, COALESCE(q.questions, '[]'::jsonb) AS questions
		FROM at_booking_people bp
		JOIN at_bookings b ON b.id=bp.booking_id
		JOIN at_trips t ON t.id=b.trip_id
		LEFT JOIN at_trip_questionnaires q ON q.trip_id=t.id AND q.revision=bp.questionnaire_revision
		WHERE bp.id=$1 AND
			(($4 AND t.owner_id=$2) OR (NOT $4 AND (b.owner_id=$2 OR $3='any')))`,
		participant, actor, scope, leader)
	if err != nil {
		return models.ParticipantDetails{}, err
	}
	if err := json.Unmarshal(row.Answers, &row.ParticipantDetails.Answers); err != nil {
		return models.ParticipantDetails{}, err
	}
	if err := json.Unmarshal(row.Questions, &row.ParticipantDetails.Questions); err != nil {
		return models.ParticipantDetails{}, err
	}
	current, err := trips.Questionnaire(ctx, s.DB, row.TripID, 0)
	if err != nil {
		return models.ParticipantDetails{}, err
	}
	row.ParticipantDetails.NeedsAnswers = current.NeedsAnswers(row.ParticipantDetails.Answers)
	return row.ParticipantDetails, nil
}

func (s Store) SaveDetails(ctx context.Context, tx *sqlx.Tx, participant int, input models.ParticipantDetails) error {
	if input.Answers == nil {
		input.Answers = map[string]json.RawMessage{}
	}
	raw, err := json.Marshal(input.Answers)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE at_booking_people
		SET submitted_name=$2, mobile_phone=$3, home_address=$4, membership_category=$5,
			emergency_contact_name=$6, emergency_contact_phone=$7, dietary_requirements=$8,
			trip_answer_history=CASE WHEN questionnaire_revision IS NOT NULL AND questionnaire_revision<>$9
				THEN trip_answer_history || jsonb_build_object(questionnaire_revision::text, trip_answers)
				ELSE trip_answer_history END,
			questionnaire_revision=NULLIF($9, 0), trip_answers=$10::jsonb
		WHERE id=$1`, participant, input.Name, input.Mobile, input.Address, input.Membership,
		input.EmergencyName, input.EmergencyPhone, input.Dietary, input.QuestionnaireRevision, string(raw))
	return changed(result, err)
}
