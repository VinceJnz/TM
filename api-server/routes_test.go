package main

import (
	"api-server/v2/app/appCore"
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
