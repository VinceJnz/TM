package handlerUser

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"api-server/v2/models"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestApprovalEmailRequiresActiveAccountAndAnyScope(t *testing.T) {
	db := testdb.Open(t, "")
	if err := migrations.Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(`INSERT INTO st_users(id,name,username,email,user_account_status_id)
		VALUES (10,'Synthetic','synthetic','synthetic@example.test',$1)`, models.AccountVerified)
	if err != nil {
		t.Fatal(err)
	}
	app := appCore.New(false)
	app.Db = db
	router := mux.NewRouter()
	New(app).RegisterRoutes(router, "/users")
	request := func(scope string, status int, body string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/users/10/approval-email", nil)
		if scope != "" {
			req = req.WithContext(context.WithValue(req.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: scope}))
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != status || !strings.Contains(res.Body.String(), body) {
			t.Fatalf("scope %q returned %d %s", scope, res.Code, res.Body.String())
		}
	}
	request("", http.StatusUnauthorized, "")
	request("own", http.StatusForbidden, "")
	request("any", http.StatusConflict, "approval_email_unavailable")
	if _, err := db.Exec("UPDATE st_users SET user_account_status_id=$1 WHERE id=10", models.AccountActive); err != nil {
		t.Fatal(err)
	}
	request("own", http.StatusForbidden, "")
	request("any", http.StatusAccepted, `"queued":true`)
	request("any", http.StatusAccepted, `"queued":false`)
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM at_notification_outbox
		WHERE account_id=10 AND kind='account-approved' AND recipient='synthetic@example.test'
		AND subject='Your Trip Manager account is approved' AND sent_at IS NULL`); err != nil || count != 1 {
		t.Fatalf("expected one queued standard email, got %d (%v)", count, err)
	}
}
