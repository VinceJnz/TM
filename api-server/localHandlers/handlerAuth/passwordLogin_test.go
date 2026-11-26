package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordSignInSendsCodeAndCreatesSession(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	const password = "synthetic-sign-in-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users (name, username, email, user_account_status_id, user_password)
		VALUES ('Synthetic Member', 'member', 'member@example.test', $1, $2)`, models.AccountActive, string(hash)); err != nil {
		t.Fatal(err)
	}
	mailbox := make(resetTestMailbox, 1)
	conf := &appCore.Config{Db: db, EmailSvc: mailbox}
	conf.Settings.DevMode = true
	conf.Settings.EmailDevModeAddr = "debug@example.test"
	handler := New(conf)
	response := httptest.NewRecorder()
	handler.LoginWithPassword(response, httptest.NewRequest(http.MethodPost, "/auth/login-password",
		strings.NewReader(`{"email":"member@example.test","user_password":"`+password+`"}`)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("password request returned %d", response.Code)
	}
	var code string
	select {
	case message := <-mailbox:
		const marker = "Your one-time password is: "
		_, tail, found := strings.Cut(message, marker)
		code, _, _ = strings.Cut(tail, "\n")
		if !strings.HasPrefix(message, "debug@example.test\n\n") || !found || code == "" {
			t.Fatal("sign-in email was missing its expected recipient or code")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no sign-in email was sent")
	}
	payload, err := json.Marshal(map[string]any{"token": code, "remember_me": true})
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.VerifyPasswordOTP(response, httptest.NewRequest(http.MethodPost, "/auth/verify-password-otp", strings.NewReader(string(payload))))
	if response.Code != http.StatusOK {
		t.Fatalf("code verification returned %d", response.Code)
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "session" && cookie.Secure && cookie.HttpOnly && !cookie.Expires.IsZero() {
			return
		}
	}
	t.Fatal("verification did not issue a persistent secure session cookie")
}
