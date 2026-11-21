package handlerOperations

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestOperationsRequiresGlobalCapabilityScopeAndReturnsAggregates(t *testing.T) {
	db := testdb.Open(t, "")
	if err := migrations.Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	app := appCore.New(false)
	app.Db = db
	for _, scope := range []string{"", "own", "any"} {
		r := httptest.NewRequest("GET", "/operations", nil)
		if scope != "" {
			r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: scope}))
		}
		w := httptest.NewRecorder()
		Read(app)(w, r)
		want := 200
		if scope == "" {
			want = 401
		}
		if scope == "own" {
			want = 403
		}
		if w.Code != want {
			t.Fatal(scope, w.Code)
		}
		if scope == "any" {
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["database_pool"] == nil || body["pending_refunds"] != float64(0) {
				t.Fatal(body, err)
			}
		}
	}
}
