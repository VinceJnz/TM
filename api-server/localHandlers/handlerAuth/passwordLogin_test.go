package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordSignInSendsCodeAndCreatesSession(t *testing.T) {
	db := testdb.Open(t, testdb.Auth+`
		CREATE TABLE st_group (id integer PRIMARY KEY, name text);
		CREATE TABLE st_user_group (user_id integer, group_id integer);
	`)
	const password = "synthetic-sign-in-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users (name, username, email, user_account_status_id, user_password)
		VALUES ('Synthetic Member', 'Member', 'member@example.test', $1, $2)`, models.AccountActive, string(hash)); err != nil {
		t.Fatal(err)
	}
	mailbox := make(resetTestMailbox, 1)
	conf := &appCore.Config{Db: db, EmailSvc: mailbox}
	conf.Settings.Host = "localhost"
	conf.Settings.DevMode = true
	conf.Settings.EmailDevModeAddr = "debug@example.test"
	handler := New(conf)
	response := httptest.NewRecorder()
	handler.LoginWithPassword(response, httptest.NewRequest(http.MethodPost, "/auth/login-password",
		strings.NewReader(`{"username":"mEmBeR","user_password":"`+password+`"}`)))
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
	const lanOrigin = "https://192.168.1.6:8086"
	verify := httptest.NewRequest(http.MethodPost, lanOrigin+"/auth/verify-password-otp", strings.NewReader(string(payload)))
	handler.VerifyPasswordOTP(response, verify)
	if response.Code != http.StatusOK {
		t.Fatalf("code verification returned %d", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].Domain != "" ||
		cookies[0].Path != "/" || !cookies[0].Secure || !cookies[0].HttpOnly ||
		cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Expires.IsZero() {
		t.Fatal("verification did not issue a persistent, host-only secure session cookie")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(verify.URL, cookies)
	menu := httptest.NewRequest(http.MethodGet, lanOrigin+"/auth/menuUser/", nil)
	for _, cookie := range jar.Cookies(menu.URL) {
		menu.AddCookie(cookie)
	}
	localhost := &url.URL{Scheme: "https", Host: "localhost:8086", Path: menu.URL.Path}
	if len(jar.Cookies(localhost)) != 0 {
		t.Fatal("LAN session was sent to localhost")
	}
	response = httptest.NewRecorder()
	handler.MenuUserGet(response, menu)
	var user models.MenuUser
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &user) != nil || user.UserID == 0 {
		t.Fatal("the menu request did not recognize the LAN session")
	}
}
