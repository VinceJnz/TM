package handlerMyProfile

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"strings"
	"testing"
)

type proofMailbox struct{ codes map[string]string }

func (m *proofMailbox) RenewURL() string { return "" }
func (m *proofMailbox) SendMail(to, subject, body string) (bool, error) {
	parts := strings.Split(body, "\n")
	m.codes[to] = parts[len(parts)-1]
	return true, nil
}
func TestEmailHTTPFlowDoesNotReturnCodesAndExpiresSessionCookie(t *testing.T) {
	db := testdb.Open(t, "")
	if err := migrations.Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','old@example.test'); INSERT INTO st_token(user_id,name,token) VALUES(1,'session','synthetic-session')`); err != nil {
		t.Fatal(err)
	}
	app := appCore.New(false)
	app.Db = db
	mailbox := &proofMailbox{codes: map[string]string{}}
	app.SetEmailService(mailbox)
	router := mux.NewRouter()
	New(app).RegisterRoutes(router, "/myProfile")
	call := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: "own"}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	requested := call("/myProfile/email-change", `{"email":"new@example.test","user_id":2}`)
	if requested.Code != 202 || len(mailbox.codes) != 2 {
		t.Fatal("verification not requested", requested.Code)
	}
	for _, code := range mailbox.codes {
		if code == "" || strings.Contains(requested.Body.String(), code) {
			t.Fatal("code disclosed by API")
		}
	}
	payload, err := json.Marshal(models.EmailChangeProof{OldCode: mailbox.codes["old@example.test"], NewCode: mailbox.codes["new@example.test"]})
	if err != nil {
		t.Fatal(err)
	}
	confirmed := call("/myProfile/email-change/confirm", string(payload))
	if confirmed.Code != 200 || !strings.Contains(confirmed.Body.String(), `"reauthentication_required":true`) {
		t.Fatal("confirmation failed", confirmed.Code)
	}
	cookies := confirmed.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].MaxAge != -1 || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatal("session cookie not securely expired")
	}
	var count int
	if err = db.Get(&count, "SELECT count(*) FROM st_token WHERE user_id=1"); err != nil || count != 0 {
		t.Fatal("session persisted", err)
	}
	if replay := call("/myProfile/email-change/confirm", string(payload)); replay.Code != 400 {
		t.Fatal("proof replay accepted", replay.Code)
	}
}
