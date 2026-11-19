package handlerBooking

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestOwnershipReads(t *testing.T) {
	db := testdb.Open(t, testdb.Bookings)
	app := &appCore.Config{Db: db, SessionIDKey: "test-session"}
	h := New(app)
	for _, tc := range []struct {
		scope, id string
		nested    bool
		wantOther bool
	}{
		{"own", "22", false, false}, {"own", "1", true, false}, {"any", "22", false, true}, {"any", "1", true, true},
	} {
		r := mux.SetURLVars(httptest.NewRequest("GET", "/", nil), map[string]string{"id": tc.id})
		r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: tc.scope}))
		w := httptest.NewRecorder()
		if tc.nested {
			h.GetList(w, r)
		} else {
			h.Get(w, r)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), `"owner_id":2`) != tc.wantOther {
			t.Fatalf("wrong ownership result: %s", w.Body.String())
		}
	}
}

func TestBookingWritesCannotChangePaymentState(t *testing.T) {
	db := testdb.Open(t, testdb.Bookings)
	// Member edits affect dates/notes, never a server price or paid state.
	if _, err := db.Exec(qryUpdate, 11, 1, "own", "updated", "2026-01-01", "2026-01-02"); err != nil {
		t.Fatal(err)
	}
	var status int
	var price string
	if err := db.QueryRow("SELECT booking_status_id,booking_price FROM at_bookings WHERE id=11").Scan(&status, &price); err != nil {
		t.Fatal(err)
	}
	if status != 1 || price != "10" {
		t.Fatal("member edit changed financial state")
	}
	for _, lock := range []string{
		"UPDATE at_bookings SET booking_status_id=3 WHERE id=11",
		"UPDATE at_bookings SET booking_status_id=1,stripe_session_id='checkout' WHERE id=11",
		"UPDATE at_bookings SET stripe_session_id=NULL WHERE id=11; INSERT INTO at_payments VALUES (1,11)",
	} {
		if _, err := db.Exec(lock); err != nil {
			t.Fatal(err)
		}
		result, err := db.Exec(qryUpdate, 11, 1, "own", "forged", "2026-01-01", "2026-01-02")
		if err != nil {
			t.Fatal(err)
		}
		rows, _ := result.RowsAffected()
		if rows != 0 {
			t.Fatal("financially locked booking was editable")
		}
		result, err = db.Exec(qryDelete, 11, 1, "any")
		if err != nil {
			t.Fatal(err)
		}
		rows, _ = result.RowsAffected()
		if rows != 0 {
			t.Fatal("financially locked booking was deletable")
		}
	}
}
