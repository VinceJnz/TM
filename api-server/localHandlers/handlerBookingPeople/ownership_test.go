package handlerBookingPeople

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestParticipantOwnershipReadsAndForgedUpdate(t *testing.T) {
	db := testdb.Open(t, testdb.Bookings)
	app := &appCore.Config{Db: db, SessionIDKey: "test-session"}
	h := New(app)
	for _, scope := range []string{"own", "any"} {
		for _, method := range []string{"all", "nested", "item"} {
			id := "222"
			if method == "nested" {
				id = "22"
			}
			r := mux.SetURLVars(httptest.NewRequest("GET", "/", nil), map[string]string{"id": id})
			r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: scope}))
			w := httptest.NewRecorder()
			switch method {
			case "all":
				h.GetAll(w, r)
			case "nested":
				h.GetList(w, r)
			case "item":
				h.Get(w, r)
			}
			if w.Code != 200 {
				t.Fatalf("%s/%s: %d %s", scope, method, w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), `"owner_id":2`) != (scope == "any") {
				t.Fatalf("ownership leak: %s", w.Body.String())
			}
		}
	}
	// A caller supplies their own booking_id while targeting someone else's participant.
	r := mux.SetURLVars(httptest.NewRequest("PUT", "/", strings.NewReader(`{"booking_id":11,"person_id":1,"notes":"forged"}`)), map[string]string{"id": "222"})
	r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: "own"}))
	w := httptest.NewRecorder()
	h.Update(w, r)
	var notes string
	if err := db.Get(&notes, "SELECT notes FROM at_booking_people WHERE id=222"); err != nil {
		t.Fatal(err)
	}
	if notes != "second" {
		t.Fatal("forged parent bypassed ownership")
	}
}
