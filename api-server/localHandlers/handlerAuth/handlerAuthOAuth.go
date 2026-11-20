package handlerAuth

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"net/http"
)

// RequireSessionAuth authenticates the session; authorization is a separate step.
func (h *Handler) RequireSessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeJSONResponse(w, 401, map[string]string{"error": "unauthorized", "message": "authentication required"}, "auth")
			return
		}
		token, err := dbAuthTemplate.FindSessionTokenContext(r.Context(), debugTag, h.appConf.Db, cookie.Value)
		if err != nil {
			code := 503
			if errors.Is(err, sql.ErrNoRows) {
				code = 401
			}
			writeJSONResponse(w, code, map[string]string{"error": "unauthorized", "message": "session unavailable or invalid"}, "auth")
			return
		}
		user, err := dbAuthTemplate.UserReadQryContext(r.Context(), debugTag, h.appConf.Db, token.UserID)
		if err != nil {
			writeJSONResponse(w, 503, map[string]string{"error": "unavailable", "message": "account lookup unavailable"}, "auth")
			return
		}
		if !user.UserActive() {
			writeJSONResponse(w, 403, map[string]string{"error": "forbidden", "message": "account is not active"}, "auth")
			return
		}
		session := &models.Session{UserID: user.ID, Email: user.Email.String}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), h.appConf.SessionIDKey, session)))
	})
}

// RequireCapability uses the matched route's declaration, never URL inference.
func (h *Handler) RequireCapability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session)
		if !ok || session == nil {
			writeJSONResponse(w, 401, map[string]string{"error": "unauthorized"}, "capability")
			return
		}
		resource, err := h.setRestResource(r)
		if err != nil {
			writeJSONResponse(w, 500, map[string]string{"error": "configuration_error", "message": "route capability is not configured"}, "capability")
			return
		}
		grant, err := dbAuthTemplate.UserCheckAccessContext(r.Context(), h.appConf.Db, session.UserID, resource.ResourceName, resource.AccessMethod)
		if err != nil {
			code := 503
			if errors.Is(err, sql.ErrNoRows) {
				code = 403
			}
			writeJSONResponse(w, code, map[string]string{"error": "forbidden", "message": "resource access unavailable or denied"}, "capability")
			return
		}
		if !supportsScope(resource.ResourceName, resource.AccessMethod, grant.AccessScope) {
			writeJSONResponse(w, 403, map[string]string{"error": "forbidden", "message": "unsupported scope"}, "capability")
			return
		}
		authorized := *session
		authorized.PrevURL = resource.PrevURL
		authorized.ResourceName = resource.ResourceName
		authorized.AccessMethod = resource.AccessMethod
		authorized.AccessScope = grant.AccessScope
		authorized.AccessScopeID = grant.AccessScopeID
		authorized.Group = grant.Group
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), h.appConf.SessionIDKey, &authorized)))
	})
}
