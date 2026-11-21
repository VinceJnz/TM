package appCore

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequiredIntegrationsAffectReadinessWithoutExternalCalls(t *testing.T) {
	db := testdb.Open(t, "")
	if err := migrations.Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	c := New(false)
	c.Db = db
	probe := func() int {
		w := httptest.NewRecorder()
		c.Readiness(w, httptest.NewRequest("GET", "/readyz", nil))
		return w.Code
	}
	if probe() != 200 {
		t.Fatal("optional integration blocked readiness")
	}
	c.Settings.RequiredIntegrations = []string{"mail"}
	if probe() != 503 || len(c.IntegrationAlerts()) != 1 {
		t.Fatal("missing mail integration reported ready")
	}
	c.SetEmailService(&capturedMailer{})
	if probe() != 200 {
		t.Fatal("initialized mail integration unavailable")
	}
	c.Settings.RequiredIntegrations = []string{"stripe", "google"}
	if probe() != 503 || !strings.Contains(strings.Join(c.IntegrationAlerts(), ","), "required_stripe_unavailable") {
		t.Fatal("missing dependencies not exposed")
	}
	if _, err := parseRequiredIntegrations("mail,typo"); err == nil {
		t.Fatal("unknown integration accepted")
	}
	if names, err := parseRequiredIntegrations(" mail,google,mail "); err != nil || len(names) != 2 {
		t.Fatal(names, err)
	}
}
