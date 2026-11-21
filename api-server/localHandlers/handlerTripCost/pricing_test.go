package handlerTripCost

import (
	"api-server/v2/app/appCore"
	"api-server/v2/models"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPricingRequiresAnyScopeAndRejectsInvalidMoney(t *testing.T) {
	app := appCore.New(false)
	h := New(app)
	for _, tc := range []struct {
		scope, body string
		code        int
	}{{"", "{}", 401}, {"own", "{}", 403}, {"any", `{"amount":-1}`, 400}, {"any", `{"amount":0.001}`, 400}, {"any", `{"amount":1}`, 400}} {
		r := httptest.NewRequest("POST", "/tripCosts", strings.NewReader(tc.body))
		if tc.scope != "" {
			r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: tc.scope}))
		}
		w := httptest.NewRecorder()
		h.Create(w, r)
		if w.Code != tc.code {
			t.Fatal(tc, w.Code)
		}
	}
}
