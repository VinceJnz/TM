package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestAuthenticationAndPermissionStatuses(t *testing.T) {
	db := testdb.Open(t, testdb.Auth+`
CREATE TABLE st_group(id integer,name text);CREATE TABLE st_user_group(user_id integer,group_id integer);
CREATE TABLE et_resource(id integer,name text);CREATE TABLE et_access_level(id integer,name text);CREATE TABLE et_access_scope(id integer,name text);
CREATE TABLE st_group_resource(group_id integer,resource_id integer,access_level_id integer,access_scope_id integer);
INSERT INTO st_users(id,name,username,email,user_account_status_id) VALUES(1,'Member','member','member@example.test',2);
INSERT INTO st_group VALUES(1,'Members');INSERT INTO st_user_group VALUES(1,1);
INSERT INTO et_resource VALUES(1,'bookings');INSERT INTO et_access_level VALUES(1,'GET');INSERT INTO et_access_scope VALUES(1,'own');`)
	h := New(&appCore.Config{Db: db, SessionIDKey: "test"})
	cookie, err := dbAuthTemplate.CreateSessionToken("test", db, 1, "localhost", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	r := mux.NewRouter()
	r.Use(h.RequireSessionAuth, h.RequireCapability)
	helpers.CapabilityRoute(r, "/custom/v9/renamed-route", "bookings", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }).Methods("GET")
	request := func(auth bool) int {
		req := httptest.NewRequest("GET", "/custom/v9/renamed-route", nil)
		if auth {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if got := request(false); got != 401 {
		t.Fatalf("anonymous status=%d", got)
	}
	if got := request(true); got != 403 {
		t.Fatalf("missing permission status=%d", got)
	}
	if _, err := db.Exec("INSERT INTO st_group_resource VALUES(1,1,1,1)"); err != nil {
		t.Fatal(err)
	}
	if got := request(true); got != 204 {
		t.Fatalf("declared capability status=%d", got)
	}
}
