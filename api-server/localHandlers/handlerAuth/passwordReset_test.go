package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"bytes"
	"encoding/base64"
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
	code := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
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
		testPasswordResetFromEmail(t, "", "member@example.test")
	})
	t.Run("development debug inbox", func(t *testing.T) {
		testPasswordResetFromEmail(t, "debug@example.test", "member@example.test", "other@example.test")
	})
}

func testPasswordResetFromEmail(t *testing.T, debugRecipient string, accountEmails ...string) {
	t.Helper()
	db := testdb.Open(t, testdb.Auth)
	for _, email := range accountEmails {
		if _, err := db.Exec(`INSERT INTO st_users (email, user_account_status_id, user_password)
			VALUES ($1, $2, 'old-hash')`, email, models.AccountActive); err != nil {
			t.Fatal(err)
		}
	}
	mailbox := make(resetTestMailbox, 1)
	router := mux.NewRouter()
	conf := &appCore.Config{Db: db, EmailSvc: mailbox}
	conf.Settings.DevMode = debugRecipient != ""
	conf.Settings.EmailDevModeAddr = debugRecipient
	New(conf).RegisterRoutesPublic(router, "/auth")
	for _, email := range accountEmails {
		t.Run(email, func(t *testing.T) {
			expectedRecipient := email
			if debugRecipient != "" {
				expectedRecipient = debugRecipient
			}
			requestPasswordReset(t, router, mailbox, email, expectedRecipient)
		})
	}
	for _, email := range accountEmails {
		var hash string
		if err := db.Get(&hash, "SELECT user_password FROM st_users WHERE email = $1", email); err != nil {
			t.Fatal(err)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(resetTestPassword)); err != nil {
			t.Fatalf("new password did not match the stored hash for %s", email)
		}
	}
}

const resetTestPassword = "synthetic-new-password"

func requestPasswordReset(t *testing.T, router *mux.Router, mailbox resetTestMailbox, email, expectedRecipient string) {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRecorder()
	router.ServeHTTP(request, httptest.NewRequest(http.MethodPost, "/auth/forgot-password",
		bytes.NewReader(payload)))
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
	payload, err = json.Marshal(map[string]string{"code": code, "password": resetTestPassword})
	if err != nil {
		t.Fatal(err)
	}
	reset := httptest.NewRecorder()
	router.ServeHTTP(reset, httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader(payload)))
	if reset.Code != http.StatusNoContent {
		t.Fatalf("password reset returned %d", reset.Code)
	}
}
