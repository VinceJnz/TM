package handlerAuth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"

	"github.com/gorilla/mux"
)

const (
	roleUser     = "user"
	roleAdmin    = "admin"
	roleSysadmin = "sysadmin"
)

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case roleSysadmin:
		return roleSysadmin
	case roleAdmin:
		return roleAdmin
	default:
		return roleUser
	}
}

func roleRank(role string) int {
	switch normalizeRole(role) {
	case roleSysadmin:
		return 3
	case roleAdmin:
		return 2
	default:
		return 1
	}
}

// The function RequireSessionAuth needs to do the following:
//1. Check to see the the users is already logged in (It already deos this)
//2. Try logging in using OAuth
//3. Try using a password and emailed token for authentication.

// RequireSessionAuth returns a middleware that accepts either a DB session cookie ("session")
// or an OAuth session ("auth-session") from the provided gateway. It ensures an internal session
// context (models.Session) is available for downstream handlers.
func (h *Handler) RequireSessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resource RestResource
		//var dbToken models.Token
		var accessCheck models.AccessCheck
		var user models.User
		//var err error
		log.Printf("%vRequireSessionAuth()0, checking auth for request: %v %v\n", debugTag, r.Method, r.URL.Path)

		// 1) Try existing DB session cookie to see if the user is already logged in
		if sessionCookie, err := r.Cookie("session"); err == nil {
			if dbToken, err := dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sessionCookie.Value); err == nil {
				// build session context (same as RequireRestAuth)

				if resource, err = h.setRestResource(r); err != nil {
					log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireSessionAuth()1", "err =", err, "resource =", resource, "r =", r)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Not authorised - You don't have access to the requested resource."))
					return
				}
				// check access to resource
				if accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireSessionAuth()1a ", h.appConf.Db, dbToken.UserID, resource.ResourceName, resource.AccessMethod); err != nil {
					log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireSessionAuth()2", "err =", err, "resource =", resource, "r =", r)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Not authorised - You don't have access to the requested resource."))
					return
				}
				// Check the user record is still valid/active
				if user, err = dbAuthTemplate.UserReadQry(debugTag+"", h.appConf.Db, dbToken.UserID); err != nil {
					log.Printf("%v failed reading user record: %v", debugTag, err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				if !user.UserActive() {
					log.Printf("%v user account not active: %v", debugTag, user.ID)
					http.Error(w, "user account not active", http.StatusInternalServerError)
					return
				}
				// 6) Attach session to context and continue
				session := &models.Session{
					UserID:         dbToken.UserID,
					PrevURL:        resource.PrevURL,
					ResourceName:   resource.ResourceName,
					ResourceID:     0,
					AccessMethod:   resource.AccessMethod,
					AccessMethodID: 0,
					AccessType:     "",
					AccessTypeID:   accessCheck.AccessTypeID,
					//AdminFlag:      accessCheck.AdminFlag,
					Role:  accessCheck.Role,
					Email: user.Email.String,
				}
				// 7) Give the user access
				ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, session)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// If DB lookup failed, continue to try OAuth session
		}
		log.Printf("%vRequireSessionAuth()3 no valid DB session, trying OAuth session\n", debugTag)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "unauthorized",
			"message": "authentication required",
		})
	})
}

func (h *Handler) RequireRole(minRole string) mux.MiddlewareFunc {
	required := roleRank(minRole)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session)
			if !ok || session == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "unauthorized",
					"message": "authentication required",
				})
				return
			}

			if roleRank(session.Role) < required {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "forbidden",
					"message": "insufficient role",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (h *Handler) RequireAdmin(next http.Handler) http.Handler {
	return h.RequireRole(roleAdmin)(next)
}

func (h *Handler) RequireSysadmin(next http.Handler) http.Handler {
	return h.RequireRole(roleSysadmin)(next)
}
