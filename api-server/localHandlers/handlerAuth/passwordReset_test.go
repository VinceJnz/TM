package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordResetRoutesValidateRequests(t *testing.T) {
	router := mux.NewRouter()
	New(nil).RegisterRoutesPublic(router, "/auth")
	code := dbAuthTemplate.GenerateSecureToken()
	for _, tc := range []struct {
		path, body string
		status     int
	}{
		{"/auth/forgot-password", `{"email":"invalid"}`, 400},
		{"/auth/forgot-password", `{"email":"member@example.test"}`, 503},
		{"/auth/reset-password", `{"code":"` + code + `","password":"short"}`, 400},
		{"/auth/reset-password", `{"code":"invalid","password":"synthetic-password"}`, 400},
		{"/auth/reset-password", `{"code":"` + code + `","password":"synthetic-password"}`, 503},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
		if w.Code != tc.status {
			t.Errorf("%s returned %d, want %d", tc.path, w.Code, tc.status)
		}
	}
}

type resetTestMailbox chan string

func (m resetTestMailbox) SendMail(to, subject, body string) (bool, error) {
	m <- to + "\n\n" + body
	return true, nil
}

func (resetTestMailbox) RenewURL() string { return "" }

func TestPasswordResetFromEmailToNewPassword(t *testing.T) {
	t.Run("registered inbox", func(t *testing.T) {
		testPasswordResetFromEmail(t, false, "member@example.test")
	})
	t.Run("development debug inbox", func(t *testing.T) {
		testPasswordResetFromEmail(t, true, "debug@example.test")
	})
}

func testPasswordResetFromEmail(t *testing.T, devMode bool, expectedRecipient string) {
	t.Helper()
	db := testdb.Open(t, testdb.Auth)
	if _, err := db.Exec(`INSERT INTO st_users (email, user_account_status_id, user_password)
		VALUES ('member@example.test', $1, 'old-hash')`, models.AccountActive); err != nil {
		t.Fatal(err)
	}
	mailbox := make(resetTestMailbox, 1)
	router := mux.NewRouter()
	conf := &appCore.Config{Db: db, EmailSvc: mailbox}
	conf.Settings.DevMode = devMode
	conf.Settings.PasswordResetTestEmail = "member@example.test"
	if devMode {
		conf.Settings.EmailDebugAddr = expectedRecipient
	}
	New(conf).RegisterRoutesPublic(router, "/auth")
	if devMode {
		blocked := httptest.NewRecorder()
		router.ServeHTTP(blocked, httptest.NewRequest(http.MethodPost, "/auth/forgot-password",
			strings.NewReader(`{"email":"other@example.test"}`)))
		if blocked.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected debug delivery to be blocked for an unconfigured email, got %d", blocked.Code)
		}
	}
	request := httptest.NewRecorder()
	router.ServeHTTP(request, httptest.NewRequest(http.MethodPost, "/auth/forgot-password",
		strings.NewReader(`{"email":"member@example.test"}`)))
	if request.Code != http.StatusAccepted {
		t.Fatalf("code request returned %d", request.Code)
	}

	var code string
	select {
	case message := <-mailbox:
		parts := strings.Split(message, "\n\n")
		if len(parts) != 4 || parts[0] != expectedRecipient {
			t.Fatal("reset email had an unexpected format or recipient")
		}
		code = parts[2]
	case <-time.After(5 * time.Second):
		t.Fatal("no reset email was sent")
	}
	if strings.Contains(request.Body.String(), code) {
		t.Fatal("reset code appeared in the API response")
	}
	const password = "synthetic-new-password"
	payload, err := json.Marshal(map[string]string{"code": code, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	reset := httptest.NewRecorder()
	router.ServeHTTP(reset, httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader(payload)))
	if reset.Code != http.StatusNoContent {
		t.Fatalf("password reset returned %d", reset.Code)
	}
	var hash string
	if err := db.Get(&hash, "SELECT user_password FROM st_users"); err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatal("new password did not match the stored hash")
	}
}
