package helpers

import (
	"api-server/v2/app/appCore"
	"api-server/v2/models"
	"context"
	"net/http/httptest"
	"testing"
)

func TestFinancialAdministrationFailsClosed(t *testing.T) {
	key := appCore.ContextKey("test")
	for _, scope := range []string{"own", "", "any"} {
		r := httptest.NewRequest("POST", "/refunds", nil)
		r = r.WithContext(context.WithValue(r.Context(), key, &models.Session{UserID: 1, AccessScope: scope}))
		w := httptest.NewRecorder()
		if RequireFinancialAccess(w, r, key) != (scope == "any") {
			t.Fatalf("incorrect access for %q", scope)
		}
	}
}
