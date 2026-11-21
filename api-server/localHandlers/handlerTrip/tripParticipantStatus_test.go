package handlerTrip

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/modelMethods/tripreport"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"strconv"
	"testing"
)

const reportFixture = `
CREATE TABLE at_trips (id int, trip_name text, from_date timestamptz, to_date timestamptz, max_participants int);
CREATE TABLE at_bookings (id int, trip_id int, owner_id int, payment_date timestamptz);
CREATE TABLE at_booking_people (id int, booking_id int, person_id int);
CREATE TABLE st_users (id int, name text, user_account_hidden bool);
INSERT INTO at_trips VALUES (1, 'First trip', now(), now(), 10), (2, 'Second trip', now(), now(), 10), (3, 'Empty trip', now(), now(), 10);
INSERT INTO at_bookings VALUES (10, 1, 1, NULL), (20, 1, 2, NULL), (30, 2, 2, NULL);
INSERT INTO at_booking_people VALUES (1, 10, 1), (2, 20, 2), (3, 30, 2);
INSERT INTO st_users VALUES (1, 'First member', true), (2, 'Second member', false);
`

func TestTripListShowsRosterAndHonorsHiddenProfiles(t *testing.T) {
	db := testdb.Open(t, reportFixture)
	roster, more, err := (tripreport.Store{DB: db}).List(context.Background(), 100, 0)
	if err != nil || more {
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

func TestTripReportHTTPPagesHaveNoResourceIDAndKeepHiddenProfiles(t *testing.T) {
	db := testdb.Open(t, reportFixture)
	app := appCore.New(false)
	app.Db = db
	router := mux.NewRouter()
	New(app).RegisterRoutes(router, "/trips")
	for offset, wantTrip := range []int{1, 1, 2, 3} {
		r := httptest.NewRequest("GET", "/tripsReport?limit=1&offset="+strconv.Itoa(offset), nil)
		r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: "any"}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal("trip report route failed", w.Code, w.Body.String())
		}
		var rows []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0]["trip_id"] != float64(wantTrip) || rows[0]["id"] != nil {
			t.Fatal(rows)
		}
		if offset == 0 && (rows[0]["person_id"] != float64(0) || rows[0]["participant_id"] != float64(0) || rows[0]["booking_position"] != float64(0) || rows[0]["booking_status"] != "" || rows[0]["person_name"] != "name hidden") {
			t.Fatal("hidden profile exposed", rows)
		}
		wantNext := ""
		if offset < 3 {
			wantNext = strconv.Itoa(offset + 1)
		}
		if w.Header().Get("X-Next-Offset") != wantNext {
			t.Fatal(w.Header())
		}
	}
	for _, query := range []string{"limit=0", "limit=501", "offset=-1", "offset=1000001"} {
		r := httptest.NewRequest("GET", "/tripsReport?"+query, nil)
		r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: "any"}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatal(query, w.Code)
		}
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := (tripreport.Store{DB: db}).List(canceled, 1, 0); err == nil {
		t.Fatal("ignored cancellation")
	}
}
