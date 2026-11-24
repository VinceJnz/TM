package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogoutRequiresPostRevokesSessionAndExpiresCookie(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	if _, err := db.Exec("INSERT INTO st_users(id,name,username,email,user_account_status_id) VALUES(1,'Synthetic','synthetic','synthetic@example.test',2)"); err != nil {
		t.Fatal(err)
	}
	app := appCore.New(false)
	app.Db = db
	app.Settings.Host = "localhost"
	h := New(app)
	cookie, err := dbAuthTemplate.CreateSessionToken("test", db, 1, "localhost", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, origin string
		userID, status int
	}{{"GET", "", 1, 405}, {"POST", "https://attacker.test", 1, 403}, {"POST", "https://localhost", 2, 503}, {"POST", "https://localhost", 1, 202}} {
		r := httptest.NewRequest(tc.method, "https://localhost/auth/logout/", nil)
		r.AddCookie(cookie)
		r.Header.Set("Origin", tc.origin)
		r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: tc.userID}))
		w := httptest.NewRecorder()
		h.AuthLogout(w, r)
		if w.Code != tc.status {
			t.Fatal(tc, w.Code)
		}
		if tc.status != 202 {
			if _, err = dbAuthTemplate.FindSessionToken("test", db, cookie.Value); err != nil {
				t.Fatal("safe request revoked session")
			}
		} else {
			cleared := w.Result().Cookies()
			if len(cleared) != 1 || cleared[0].MaxAge >= 0 || !cleared[0].Secure || !cleared[0].HttpOnly || cleared[0].SameSite != http.SameSiteLaxMode {
				t.Fatal(cleared)
			}
			if _, err = dbAuthTemplate.FindSessionToken("test", db, cookie.Value); err == nil {
				t.Fatal("session retained")
			}
		}
	}
}
