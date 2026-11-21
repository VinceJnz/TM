package helpers

import (
	"api-server/v2/app/appCore"
	"api-server/v2/models"
	"net/http"
)

func RequiredSession(w http.ResponseWriter, r *http.Request, key appCore.ContextKey) *models.Session {
	session, ok := r.Context().Value(key).(*models.Session)
	if !ok || session == nil || session.UserID <= 0 {
		WriteError(w, 401, "unauthorized", "Authentication required")
		return nil
	}
	return session
}
func RouteID(w http.ResponseWriter, r *http.Request) int {
	id, err := GetIDFromRequest(r)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid record ID", 400)
		return 0
	}
	return id
}
