package handlerGmailCert

import (
	"api-server/v2/app/appCore"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

func TestGatewayRoutesDeclareCredentialCapability(t *testing.T) {
	router := mux.NewRouter()
	New(appCore.New(false)).RegisterRoutes(router, "/gmailcert")
	for _, tc := range []struct{ method, path string }{
		{"GET", "/gmailcert"}, {"PUT", "/gmailcert"}, {"PUT", "/gmailcert/refresh"},
		{"GET", "/gmailcert/renew-url"}, {"GET", "/gmailcert/callback"},
	} {
		var match mux.RouteMatch
		if !router.Match(httptest.NewRequest(tc.method, tc.path, nil), &match) ||
			match.Route.GetName() != "capability:gmailcert" {
			t.Errorf("%s %s lacks the gmailcert capability", tc.method, tc.path)
		}
	}
}

func TestInvalidUploadPreservesSavedCredentials(t *testing.T) {
	config := appCore.New(false)
	config.Settings.EmailSecret = filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(config.Settings.EmailSecret, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	h := New(config)
	result := httptest.NewRecorder()
	h.Update(result, httptest.NewRequest(http.MethodPut, "/gmailcert", strings.NewReader(
		`{"type":"secret","content":"{}"}`,
	)))
	if result.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", result.Code, http.StatusBadRequest)
	}
	contents, err := os.ReadFile(config.Settings.EmailSecret)
	if err != nil || string(contents) != "original" {
		t.Fatalf("saved credential changed or could not be read: %v", err)
	}
}

type tokenTransport func(*http.Request) (*http.Response, error)

func (transport tokenTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return transport(r)
}

func TestRefreshOnlyReplacesTokenAfterSuccessfulExchange(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		name := "rejected"
		if accepted {
			name = "accepted"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			config := appCore.New(false)
			config.Settings.EmailSecret = filepath.Join(dir, "client.json")
			config.Settings.EmailToken = filepath.Join(dir, "token.json")
			original := `{"access_token":"previous","refresh_token":"saved","expiry":"2099-01-01T00:00:00Z"}`
			credentials := `{"web":{"client_id":"client","client_secret":"placeholder","redirect_uris":["https://example.invalid/callback"]}}`
			for path, content := range map[string]string{config.Settings.EmailSecret: credentials, config.Settings.EmailToken: original} {
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			called := false
			client := &http.Client{Transport: tokenTransport(func(r *http.Request) (*http.Response, error) {
				called = true
				if err := r.ParseForm(); err != nil || r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "saved" {
					t.Error("expected a refresh token exchange")
				}
				status, body := http.StatusBadRequest, `{"error":"invalid_grant"}`
				if accepted {
					status, body = http.StatusOK, `{"access_token":"replacement","token_type":"Bearer","expires_in":3600}`
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
			request := httptest.NewRequest(http.MethodPut, "/gmailcert/refresh", nil).WithContext(ctx)
			result := httptest.NewRecorder()
			New(config).Refresh(result, request)
			if !called {
				t.Fatal("a valid cached access token skipped the exchange")
			}
			contents, err := os.ReadFile(config.Settings.EmailToken)
			if err != nil {
				t.Fatal(err)
			}
			if accepted {
				if result.Code != http.StatusOK || !strings.Contains(string(contents), "replacement") || config.EmailService() == nil {
					t.Fatalf("successful refresh did not install the new token, status %d", result.Code)
				}
			} else if result.Code != http.StatusBadGateway || string(contents) != original || config.EmailService() != nil {
				t.Fatalf("failed refresh changed the saved token or service, status %d", result.Code)
			}
		})
	}
}
