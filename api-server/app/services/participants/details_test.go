package participants

import (
	bookingService "api-server/v2/app/services/bookings"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	bookingStore "api-server/v2/modelMethods/bookings"
	"api-server/v2/modelMethods/participants"
	"api-server/v2/modelMethods/paymentLedger"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/modelMethods/trips"
	"api-server/v2/models"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestQuestionnaireRevisionAndParticipantDetails(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(`
		INSERT INTO st_users (id,name,username,email) VALUES
			(1,'Leader','leader','leader@example.test'),(2,'Booker','booker','booker@example.test'),
			(3,'Other','other','other@example.test');
		INSERT INTO at_trips (id,owner_id,trip_name,trip_type_id,trip_status_id,max_participants)
			VALUES (1,1,'Fixture',1,2,10);
		INSERT INTO at_bookings (id,trip_id,owner_id,booking_status_id,from_date,to_date)
			VALUES (1,1,2,1,'2026-12-01','2026-12-02');
	`)
	if err != nil {
		t.Fatal(err)
	}
	tripStore := trips.Store{DB: db}
	people := participants.Store{DB: db}
	service := Service{Store: people}
	q := models.Questionnaire{Questions: []models.Question{{Key: "stayed_before", Label: "Stayed before?", Type: "yes_no", Required: true}}}
	if _, err := tripStore.PublishQuestionnaire(ctx, 1, 3, false, q); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unrelated user could publish questionnaire: %v", err)
	}
	q, err = tripStore.PublishQuestionnaire(ctx, 1, 1, false, q)
	if err != nil || q.Revision != 1 {
		t.Fatalf("publish revision one: %v %v", q, err)
	}
	input := models.ParticipantWrite{BookingID: 1, PersonID: 2, Details: &models.ParticipantDetails{
		Name: "Booker", Mobile: "0200000000", Address: "Synthetic address", Membership: "member",
		EmergencyName: "Test Contact", EmergencyPhone: "0200000001", Dietary: "None", QuestionnaireRevision: q.Revision,
	}}
	if _, err := service.Mutate(ctx, 2, "own", 0, input, false); !errors.Is(err, models.ErrQuestionnaire) {
		t.Fatalf("required answer was bypassed: %v", err)
	}
	if _, err := paymentLedger.ReviewBooking(ctx, db, 1, 2); !errors.Is(err, paymentLedger.ErrParticipantDetails) {
		t.Fatalf("checkout review accepted a booking without participants: %v", err)
	}
	input.Details.Answers = map[string]json.RawMessage{"stayed_before": json.RawMessage("false")}
	participant, err := service.Mutate(ctx, 2, "own", 0, input, false)
	if err != nil {
		t.Fatal(err)
	}
	replacement := input
	replacement.PersonID = 3
	replacement.Details = nil
	if _, err := service.Mutate(ctx, 2, "own", participant, replacement, false); !errors.Is(err, ErrValidation) {
		t.Fatalf("participant replaced without fresh details: %v", err)
	}
	if _, err := people.Details(ctx, participant, 3, "own", false); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unrelated user could read private details: %v", err)
	}
	if _, err := people.Details(ctx, participant, 3, "any", true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unrelated roster user could read private details: %v", err)
	}
	q.Questions = append(q.Questions, models.Question{Key: "equipment", Label: "Equipment needed?", Type: "yes_no", Required: true})
	q, err = tripStore.PublishQuestionnaire(ctx, 1, 1, false, q)
	if err != nil || q.Revision != 2 {
		t.Fatalf("publish revision two: %v %v", q, err)
	}
	if _, err := service.Mutate(ctx, 2, "own", participant, input, false); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatalf("obsolete questionnaire was accepted: %v", err)
	}
	saved, err := people.Details(ctx, participant, 1, "any", true)
	if err != nil || !saved.NeedsAnswers || saved.QuestionnaireRevision != 1 || string(saved.Answers["stayed_before"]) != "false" {
		t.Fatalf("original answer or required-answer indicator lost: %+v %v", saved, err)
	}
	if _, err := paymentLedger.ReviewBooking(ctx, db, 1, 2); !errors.Is(err, paymentLedger.ErrParticipantDetails) {
		t.Fatalf("checkout review accepted incomplete current answers: %v", err)
	}
	fresh := *input.Details
	fresh.Name = "Other"
	fresh.QuestionnaireRevision = q.Revision
	fresh.Answers = map[string]json.RawMessage{"stayed_before": json.RawMessage("true"), "equipment": json.RawMessage("false")}
	replacement.Details = &fresh
	if _, err := service.Mutate(ctx, 2, "own", participant, replacement, false); err != nil {
		t.Fatalf("replace participant with fresh details: %v", err)
	}
	record, err := people.Get(ctx, 2, "own", participant)
	if err != nil || record.PersonID != 3 {
		t.Fatalf("replacement person was not saved: %+v %v", record, err)
	}
	saved, err = people.Details(ctx, participant, 2, "own", false)
	if err != nil || saved.Name != "Other" || string(saved.Answers["stayed_before"]) != "true" {
		t.Fatalf("replacement details were not saved: %+v %v", saved, err)
	}
	if _, err := paymentLedger.ReviewBooking(ctx, db, 1, 2); !errors.Is(err, paymentLedger.ErrBookingPricing) {
		t.Fatalf("missing trip rate was not reported: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO at_trips (id,owner_id,trip_name,trip_type_id,trip_status_id,max_participants,from_date,to_date)
			VALUES (2,1,'New trip',1,2,10,'2026-12-01','2026-12-03')`); err != nil {
		t.Fatal(err)
	}
	choices, _, err := tripStore.BookingChoices(ctx, resourcequery.Page{Limit: 10})
	if err != nil || len(choices) != 1 || choices[0].ID != 2 {
		t.Fatalf("trip lookup should skip the undated trip: %+v %v", choices, err)
	}
	bookings := bookingService.Service{Store: bookingStore.Store{DB: db}, AllowTripChange: true}
	change := models.Booking{ID: 1, FromDate: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		ToDate: time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC)}
	change.TripID.Int64, change.TripID.Valid = 2, true
	if _, err := bookings.Save(ctx, 2, false, "update", change); err != nil {
		t.Fatalf("change draft trip: %v", err)
	}
	var archived bool
	if err := db.Get(&archived, `SELECT questionnaire_revision IS NULL AND trip_answers='{}'::jsonb
		AND trip_answer_history->'1:2'->>'equipment'='false' FROM at_booking_people WHERE id=$1`, participant); err != nil || !archived {
		t.Fatalf("old answers were not archived with the previous trip: %v, %v", archived, err)
	}
	stored, err := people.Questionnaire(ctx, 1, 2, "own")
	if err != nil || stored.TripID != 2 {
		t.Fatalf("booking did not move to the selected trip: %+v %v", stored, err)
	}
	if _, err := bookings.Save(ctx, 2, false, "delete", models.Booking{ID: 1}); err != nil {
		t.Fatalf("cancel draft with a participant: %v", err)
	}
	if _, err := people.Get(ctx, 2, "own", participant); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cancelled draft kept participant row: %v", err)
	}
}
