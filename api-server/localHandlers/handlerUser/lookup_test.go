package handlerUser

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestLookupRequiresAnyScopeAndReturnsBoundedPublicFields(t *testing.T) {
	db := testdb.Open(t, `CREATE TABLE st_users(id integer,name text,username text,email text);INSERT INTO st_users SELECT n,'Person '||n,'person'||n,'private@example.test' FROM generate_series(1,40)n;`)
	app := appCore.New(false)
	app.Db = db
	handler := New(app)
	for _, scope := range []string{"own", "any"} {
		req := httptest.NewRequest("GET", "/users/lookup?q=person", nil)
		req = req.WithContext(context.WithValue(req.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: scope}))
		response := httptest.NewRecorder()
		handler.Lookup(response, req)
		if scope == "own" {
			if response.Code != 403 {
				t.Fatal(response.Code)
			}
			continue
		}
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
		var records []map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &records); err != nil {
			t.Fatal(err)
		}
		if len(records) != 25 || len(records[0]) != 3 || records[0]["email"] != nil {
			t.Fatal("lookup unbounded or exposed account data", records)
		}
	}
}
