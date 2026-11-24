package trips

import (
	"api-server/v2/models"
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"
)

func Questionnaire(ctx context.Context, db sqlx.QueryerContext, trip, owner int) (models.Questionnaire, error) {
	var result models.Questionnaire
	var row struct {
		Revision  int    `db:"revision"`
		Questions []byte `db:"questions"`
	}
	err := sqlx.GetContext(ctx, db, &row, `
		SELECT COALESCE(q.revision, 0) AS revision,
		       COALESCE(q.questions, tt.default_questions, '[]'::jsonb) AS questions
		FROM at_trips t
		LEFT JOIN et_trip_type tt ON tt.id=t.trip_type_id
		LEFT JOIN LATERAL (
			SELECT revision, questions FROM at_trip_questionnaires
			WHERE trip_id=t.id ORDER BY revision DESC LIMIT 1
		) q ON true
		WHERE t.id=$1 AND ($2=0 OR t.owner_id=$2)`, trip, owner)
	if err != nil {
		return result, err
	}
	result.Revision = row.Revision
	err = json.Unmarshal(row.Questions, &result.Questions)
	return result, err
}

func (s Store) PublishQuestionnaire(ctx context.Context, trip, actor int, anyScope bool, input models.Questionnaire) (models.Questionnaire, error) {
	if err := input.Validate(); err != nil {
		return models.Questionnaire{}, err
	}
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return models.Questionnaire{}, err
	}
	defer tx.Rollback()
	if err := s.Lock(ctx, tx, trip, actor, anyScope); err != nil {
		return models.Questionnaire{}, err
	}
	previous, err := Questionnaire(ctx, tx, trip, 0)
	if err != nil {
		return models.Questionnaire{}, err
	}
	if input.Revision != previous.Revision {
		return models.Questionnaire{}, models.ErrStaleRecord
	}
	input.Revision++
	if err := InsertQuestionnaire(ctx, tx, trip, input); err != nil {
		return models.Questionnaire{}, err
	}
	return input, tx.Commit()
}

func InsertQuestionnaire(ctx context.Context, tx *sqlx.Tx, trip int, questionnaire models.Questionnaire) error {
	raw, err := json.Marshal(questionnaire.Questions)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO at_trip_questionnaires (trip_id, revision, questions)
		VALUES ($1,$2,$3::jsonb)`, trip, questionnaire.Revision, string(raw))
	return err
}
