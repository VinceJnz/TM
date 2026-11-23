package main

import (
	"api-server/v2/app/appCore"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestEveryProtectedRouteDeclaresItsCapability(t *testing.T) {
	app := appCore.New(false)
	app.Settings.APIprefix = "/custom/v9"
	r := buildRouter(app)
	count := 0
	err := r.Walk(func(route *mux.Route, _ *mux.Router, ancestors []*mux.Route) error {
		protected := false
		for _, parent := range ancestors {
			if parent.GetName() == "protected-api" {
				protected = true
			}
		}
		if protected {
			count++
			if !strings.HasPrefix(route.GetName(), "capability:") {
				path, _ := route.GetPathTemplate()
				t.Errorf("missing capability on %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 100 {
		t.Fatalf("only %d protected routes checked", count)
	}
}

func TestPublicAndSessionRouteInventory(t *testing.T) {
	app := appCore.New(false)
	app.Settings.APIprefix = "/custom/v9"
	router := buildRouter(app)
	expected := map[string]string{
		"GET /auth/menuUser/":                    "public-api",
		"GET /auth/content/{page}/":              "public-api",
		"GET /auth/status":                       "public-api",
		"POST /auth/register":                    "public-api",
		"POST /auth/verify-registration":         "public-api",
		"POST /auth/login":                       "public-api",
		"POST /auth/verify-otp":                  "public-api",
		"POST /auth/login-password":              "public-api",
		"POST /auth/verify-password-otp":         "public-api",
		"POST /auth/forgot-password":             "public-api",
		"POST /auth/reset-password":              "public-api",
		"POST /auth/requestToken/":               "public-api",
		"GET /auth/oauth/debug":                  "public-api",
		"GET /auth/oauth/login":                  "public-api",
		"GET /auth/oauth/callback":               "public-api",
		"GET /auth/oauth/verify-email":           "public-api",
		"POST /auth/oauth/verify-email":          "public-api",
		"GET /auth/oauth/logout":                 "public-api",
		"GET /auth/oauth/me":                     "public-api",
		"POST /auth/oauth/complete-registration": "public-api",
		"POST /bookings/checkout/webhook":        "public-api",
		"GET /userAgeGroups":                     "public-api",
		"GET /userAgeGroups/{id:[0-9]+}":         "public-api",
		"GET /auth/menuList/":                    "session-api",
		"GET /auth/logout/":                      "session-api",
		"POST /auth/logout/":                     "session-api",
	}
	err := router.Walk(func(route *mux.Route, _ *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil || !strings.HasPrefix(path, app.Settings.APIprefix+"/") {
			return nil
		}
		group := ""
		if len(ancestors) > 0 {
			group = ancestors[0].GetName()
		}
		if group == "protected-api" {
			return nil
		}
		methods, err := route.GetMethods()
		if err != nil {
			t.Errorf("route %s has no declared methods", path)
			return nil
		}
		for _, method := range methods {
			key := method + " " + strings.TrimPrefix(path, app.Settings.APIprefix)
			if want, listed := expected[key]; !listed || group != want {
				t.Errorf("unexpected %s in route group %q", key, group)
				continue
			}
			delete(expected, key)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for key, group := range expected {
		t.Errorf("missing %s in route group %q", key, group)
	}
}

func TestNestedAndAliasCapabilities(t *testing.T) {
	app := appCore.New(false)
	app.Settings.APIprefix = "/custom/v9"
	router := buildRouter(app)
	for _, tc := range []struct{ method, path, resource string }{
		{"GET", "/trips/10/bookings", "bookings"},
		{"POST", "/bookings/checkout/confirm/10", "bookings"},
		{"PUT", "/securityAccessType/10", "securityAccessScope"},
	} {
		req := httptest.NewRequest(tc.method, app.Settings.APIprefix+tc.path, nil)
		var match mux.RouteMatch
		if !router.Match(req, &match) || match.Route.GetName() != "capability:"+tc.resource {
			t.Errorf("%s %s did not declare %s", tc.method, tc.path, tc.resource)
			continue
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session returned %d", tc.method, tc.path, response.Code)
		}
	}
}
