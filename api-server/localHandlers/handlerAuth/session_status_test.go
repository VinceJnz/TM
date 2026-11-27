package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthStatusReflectsSessionAndAccountState(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email,user_account_status_id)
		VALUES(1,'Synthetic','synthetic','synthetic@example.test',2)`); err != nil {
		t.Fatal(err)
	}
	app := appCore.New(false)
	app.Db = db
	h := New(app)
	cookie := issueTestSession(t, h)

	checkStatus := func(session *http.Cookie, authenticated bool) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
		if session != nil {
			request.AddCookie(session)
		}
		response := httptest.NewRecorder()
		h.AuthStatus(response, request)
		var status struct {
			Authenticated bool `json:"authenticated"`
			User          *struct {
				ID int `json:"user_id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || status.Authenticated != authenticated {
			t.Fatalf("status response was HTTP %d, authenticated=%v", response.Code, status.Authenticated)
		}
		if authenticated && (status.User == nil || status.User.ID != 1) {
			t.Fatal("active session did not return the account")
		}
		if !authenticated && status.User != nil {
			t.Fatal("unauthenticated status returned a user")
		}
	}
	checkStatus(nil, false)
	checkStatus(cookie, true)
	if _, err := db.Exec("UPDATE st_users SET user_account_status_id=1 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	checkStatus(cookie, false)
}
