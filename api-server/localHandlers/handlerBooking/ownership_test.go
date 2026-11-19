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
