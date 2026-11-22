package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/localHandlers/handlerMyProfile"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestOwnProfileAccessRequiresGroupGrants(t *testing.T) {
	db := testdb.Open(t, testdb.Auth+`
ALTER TABLE st_users ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;CREATE TABLE st_group(id integer, name text);
CREATE TABLE et_user_age_groups(id integer,age_group text);
CREATE TABLE st_user_group(user_id integer, group_id integer);
CREATE TABLE et_resource(id integer, name text);
CREATE TABLE et_access_level(id integer, name text);
CREATE TABLE et_access_scope(id integer, name text);
ALTER TABLE st_users ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;CREATE TABLE st_group_resource(group_id integer, resource_id integer, access_level_id integer, access_scope_id integer);
INSERT INTO st_users(id, name, username, email, user_account_status_id) VALUES
	(1, 'Member', 'member', 'member@example.test', 2),
	(2, 'Other', 'other', 'other@example.test', 2);
INSERT INTO st_group VALUES (1, 'Members');
INSERT INTO et_resource VALUES (1, 'myProfile');
INSERT INTO et_access_level VALUES (1, 'GET'), (2, 'PUT');
INSERT INTO et_access_scope VALUES (1, 'own');
INSERT INTO st_group_resource VALUES (1, 1, 1, 1);
`)
	conf := &appCore.Config{Db: db, SessionIDKey: "profile-test"}
	auth := New(conf)
	cookie, err := dbAuthTemplate.CreateSessionToken("test", db, 1, "localhost", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE st_token SET valid_from = NOW() - INTERVAL '1 minute', valid_to = NOW() + INTERVAL '1 hour'`); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	auth.RegisterRoutesPublic(router, "/auth")
	sessionRoutes := router.NewRoute().Subrouter()
	sessionRoutes.Use(auth.RequireSessionAuth)
	auth.RegisterRoutesSession(sessionRoutes, "/auth")
	protected := router.NewRoute().Subrouter()
	protected.Use(auth.RequireSessionAuth, auth.RequireCapability)
	handlerMyProfile.New(conf).RegisterRoutes(protected, "/myProfile")

	call := func(method, path, body string, session *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if session != nil {
			r.AddCookie(session)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	menu := call(http.MethodGet, "/auth/menuUser/", "", cookie)
	var user models.MenuUser
	if err := json.Unmarshal(menu.Body.Bytes(), &user); err != nil || user.UserID != 1 || user.Group != "" {
		t.Fatalf("group-less session was not recognized: status %d, decode error %v", menu.Code, err)
	}
	if get := call(http.MethodGet, "/myProfile", "", cookie); get.Code != http.StatusForbidden {
		t.Fatalf("profile GET without membership returned %d", get.Code)
	}
	if _, err := db.Exec(`INSERT INTO st_user_group VALUES (1, 1)`); err != nil {
		t.Fatal(err)
	}
	capabilities := call(http.MethodGet, "/auth/menuList/", "", cookie)
	var items []models.MenuItem
	if err := json.Unmarshal(capabilities.Body.Bytes(), &items); err != nil ||
		len(items) != 1 || items[0].Name != "myProfile" || items[0].AccessLevel != http.MethodGet {
		t.Fatalf("profile grant was not listed: status %d, decode error %v", capabilities.Code, err)
	}
	if get := call(http.MethodGet, "/myProfile", "", cookie); get.Code != http.StatusOK {
		t.Fatalf("profile GET returned %d", get.Code)
	}
	const update = `{"version":1,"id":2,"name":"Updated","email":"member@example.test","user_birth_date":"2000-01-01","user_account_hidden":true}`
	if put := call(http.MethodPut, "/myProfile", update, nil); put.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous profile PUT returned %d", put.Code)
	}
	if put := call(http.MethodPut, "/myProfile", update, cookie); put.Code != http.StatusForbidden {
		t.Fatalf("profile PUT with only a GET grant returned %d", put.Code)
	}
	if _, err := db.Exec(`INSERT INTO st_group_resource VALUES (1, 1, 2, 1)`); err != nil {
		t.Fatal(err)
	}
	if put := call(http.MethodPut, "/myProfile", update, cookie); put.Code != http.StatusOK {
		t.Fatalf("profile PUT returned %d", put.Code)
	}
	if put := call(http.MethodPut, "/myProfile", strings.Replace(update, "member@example.test", "unverified@example.test", 1), cookie); put.Code != http.StatusConflict {
		t.Fatalf("unverified profile email change returned %d", put.Code)
	}
	for _, path := range []string{"/myProfile/email-change", "/myProfile/email-change/confirm"} {
		if got := call(http.MethodPost, path, `{}`, nil); got.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous nested route returned %d", got.Code)
		}
		if got := call(http.MethodPost, path, `{}`, cookie); got.Code != http.StatusForbidden {
			t.Fatalf("nested route without POST grant returned %d", got.Code)
		}
	}
	var names []string
	if err := db.Select(&names, `SELECT name FROM st_users ORDER BY id`); err != nil ||
		len(names) != 2 || names[0] != "Updated" || names[1] != "Other" {
		t.Fatalf("profile update must modify only the session user: %v", err)
	}
}
