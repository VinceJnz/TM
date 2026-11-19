package handlerOAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	"api-server/v2/internal/testdb"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestCallbackRejectsMissingOrMismatchedBrowserCookie(t *testing.T) {
	for _, value := range []string{"", "another-browser"} {
		r := httptest.NewRequest("GET", "/callback?state=expected&code=test", nil)
		if value != "" {
			r.AddCookie(&http.Cookie{Name: "oauth-state", Value: value})
		}
		w := httptest.NewRecorder()
		(&Handler{}).callbackHandler(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatal("callback did not enforce browser binding")
		}
	}
}

func TestOAuthPKCEAndSingleUseState(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	var challenge string
	exchanges := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			exchanges++
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if verifier := r.Form.Get("code_verifier"); verifier == "" || oauth2.S256ChallengeFromVerifier(verifier) != challenge {
				t.Error("PKCE verifier did not match challenge")
			}
			w.Write([]byte(`{"access_token":"synthetic-provider-token","token_type":"Bearer"}`))
			return
		}
		w.Write([]byte(`{"sub":"synthetic-subject","email":"member@example.test","name":"Member","email_verified":true}`))
	}))
	defer provider.Close()
	app := &appCore.Config{Db: db, OAuthSvc: &oAuthGateway.Gateway{OAuthConfig: &oauth2.Config{ClientID: "test", Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token", AuthStyle: oauth2.AuthStyleInParams}}, ClientRedirect: "https://app.example.test"}}
	h := New(app)
	h.userInfoURL = provider.URL + "/identity"
	start := httptest.NewRecorder()
	h.loginHandler(start, httptest.NewRequest("GET", "/login", nil))
	if start.Code != 302 {
		t.Fatalf("login: %d %s", start.Code, start.Body.String())
	}
	location, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	challenge = location.Query().Get("code_challenge")
	if challenge == "" || location.Query().Get("code_challenge_method") != "S256" {
		t.Fatal("missing PKCE")
	}
	cookie := start.Result().Cookies()[0]
	callback := func(cookieValue string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/callback?state="+url.QueryEscape(cookie.Value)+"&code=synthetic-code", nil)
		r.AddCookie(&http.Cookie{Name: "oauth-state", Value: cookieValue})
		w := httptest.NewRecorder()
		h.callbackHandler(w, r)
		return w
	}
	if callback("wrong-browser").Code != 403 {
		t.Fatal("mismatched browser accepted")
	}
	w := callback(cookie.Value)
	if w.Code != 302 {
		t.Fatalf("callback failed: %d %s", w.Code, w.Body.String())
	}
	var stateCount, continuationCount int
	if err := db.Get(&stateCount, "SELECT count(*) FROM st_token WHERE name='oauth-state'"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&continuationCount, "SELECT count(*) FROM st_token WHERE name='oauth-registration'"); err != nil {
		t.Fatal(err)
	}
	if stateCount != 0 || continuationCount != 1 {
		t.Fatal("callback state was retained or reused for registration")
	}
	if callback(cookie.Value).Code != 403 || exchanges != 1 {
		t.Fatal("OAuth callback was replayable")
	}
}
