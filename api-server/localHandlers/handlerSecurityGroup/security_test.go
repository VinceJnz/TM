package handlerSecurityGroup

import (
	"api-server/v2/app/appCore"
	"api-server/v2/models"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityWritesRequireGlobalScope(t *testing.T) {
	app := appCore.New(false)
	h := New(app)
	for _, scope := range []string{"", "own", "any"} {
		r := httptest.NewRequest("POST", "/securityGroup", strings.NewReader("{}"))
		if scope != "" {
			r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: scope}))
		}
		w := httptest.NewRecorder()
		h.Create(w, r)
		want := 422
		if scope == "" {
			want = 401
		}
		if scope == "own" {
			want = 403
		}
		if w.Code != want {
			t.Fatal(scope, w.Code)
		}
	}
}
