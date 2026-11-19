package helpers

import (
	"api-server/v2/app/appCore"
	"api-server/v2/models"
	"net/http"
	"strings"
)

// Financial administration must never let a member credit or refund their own
// booking. Own-scope grants are unsupported here and fail closed, including reads.
func RequireFinancialAccess(w http.ResponseWriter, r *http.Request, key appCore.ContextKey) bool {
	session, ok := r.Context().Value(key).(*models.Session)
	if !ok || session == nil || session.UserID <= 0 {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(session.AccessScope), "any") {
		WriteError(w, http.StatusForbidden, "forbidden", "financial administration requires any scope")
		return false
	}
	return true
}
