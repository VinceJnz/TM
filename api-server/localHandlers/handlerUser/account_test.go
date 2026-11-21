package handlerUser

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestAccountHTTPRequiresGlobalScopeAndReasonAndIgnoresSecretFields(t *testing.T) {
	db := testdb.Open(t, "")
	if err := migrations.Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	app := appCore.New(false)
	app.Db = db
	router := mux.NewRouter()
	New(app).RegisterRoutes(router, "/users")
	call := func(method, path, body, scope, reason string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("X-Change-Reason", reason)
		if scope != "" {
			r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 100, AccessScope: scope}))
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	body := `{"name":"Synthetic","username":"synthetic","email":"synthetic@example.test","user_password":"injected","provider_id":"injected","user_account_status_id":2}`
	if w := call("POST", "/users", body, "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/users", body, "own", ""); w.Code != 403 {
		t.Fatal(w.Code)
	}
	created := call("POST", "/users", body, "any", "")
	if created.Code != 201 {
		t.Fatal(created.Code, created.Body.String())
	}
	var id int
	if err := json.Unmarshal(created.Body.Bytes(), &id); err != nil || id <= 0 {
		t.Fatal("invalid created ID", err)
	}
	path := "/users/" + strconv.Itoa(id)
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM st_users WHERE user_password IS NOT NULL OR provider_id IS NOT NULL`); err != nil || count != 0 {
		t.Fatal("injected credentials stored", err)
	}
	update := strings.Replace(body, "synthetic@example.test", "new@example.test", 1)
	if w := call("PUT", path, update, "any", ""); w.Code != 422 {
		t.Fatal("reason not required", w.Code)
	}
	if w := call("PUT", path, update, "any", "verified recovery"); w.Code != 200 || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "provider_id") {
		t.Fatal("unsafe account response", w.Code)
	}
	if w := call("GET", path, "", "own", ""); w.Code != 403 {
		t.Fatal("own read allowed", w.Code)
	}
	if w := call("DELETE", path, "", "any", ""); w.Code != 422 {
		t.Fatal("delete without reason", w.Code)
	}
}
