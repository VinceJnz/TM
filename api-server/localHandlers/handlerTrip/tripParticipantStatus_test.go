package handlerTrip

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"testing"
)

func TestTripListShowsRosterAndHonorsHiddenProfiles(t *testing.T) {
	db := testdb.Open(t, `
CREATE TABLE at_trips (id int, trip_name text, from_date timestamptz, to_date timestamptz, max_participants int);
CREATE TABLE at_bookings (id int, trip_id int, owner_id int, payment_date timestamptz);
CREATE TABLE at_booking_people (id int, booking_id int, person_id int);
CREATE TABLE st_users (id int, name text, user_account_hidden bool);
INSERT INTO at_trips VALUES (1, 'First trip', now(), now(), 10), (2, 'Second trip', now(), now(), 10), (3, 'Empty trip', now(), now(), 10);
INSERT INTO at_bookings VALUES (10, 1, 1, NULL), (20, 1, 2, NULL), (30, 2, 2, NULL);
INSERT INTO at_booking_people VALUES (1, 10, 1), (2, 20, 2), (3, 30, 2);
INSERT INTO st_users VALUES (1, 'First member', true), (2, 'Second member', false);
`)
	var roster []models.TripParticipantStatus
	if err := db.Select(&roster, sqlGetParticipantStatus); err != nil {
		t.Fatal(err)
	}
	if len(roster) != 4 || roster[3].TripID != 3 || roster[3].BookingID.Valid {
		t.Fatal("expected all three trips, including the trip with no bookings")
	}
	hidden := roster[0]
	if hidden.PersonName.String != "name hidden" || hidden.PersonID.Valid || hidden.ParticipantID.Valid ||
		hidden.BookingPosition.Valid || hidden.BookingStatus.Valid {
		t.Fatal("hidden profile exposed participant details")
	}
	visible := roster[1]
	if visible.BookingID.Int64 != 20 || visible.PersonName.String != "Second member" ||
		visible.BookingPosition.Int64 != 2 || visible.BookingStatus.String != "before_threshold" ||
		roster[2].BookingID.Int64 != 30 {
		t.Fatal("visible participant details and trip order should be preserved")
	}
}
