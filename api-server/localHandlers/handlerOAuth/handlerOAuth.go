package handlerOAuth

import (
	"context"
	"log"
	"net/http"
	"time"

	"api-server/v2/app/appCore"
	oauthgw "api-server/v2/app/gateways/oAuthGoogle/oAuthGateway"
	"api-server/v2/app/srpPool"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerOAuth."

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Handler struct {
	appConf *appCore.Config
	Pool    *srpPool.Pool
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
		Pool:    srpPool.NewSRPPool(),
	}
}

// RequireOAuthOrSessionAuth returns a middleware that accepts either a DB session cookie ("session")
// or an OAuth session ("auth-session") from the provided gateway. It ensures an internal session
// context (models.Session) is available for downstream handlers.
func (h *Handler) RequireOAuthOrSessionAuth(gw *oauthgw.Gateway) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1) Try existing DB session cookie
			if sc, err := r.Cookie("session"); err == nil {
				if token, err := dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sc.Value); err == nil {
					// build session context (same as RequireRestAuth)
					sess := &models.Session{UserID: token.UserID}
					ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, sess)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// If DB lookup failed, continue to try OAuth session
			}

			// 2) Try OAuth session (gorilla/sessions store)
			oauthSess, _ := gw.Store.Get(r, "auth-session")
			providerID, _ := oauthSess.Values["user_id"].(string)
			if providerID == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// 3) Find or create an internal user record mapped to this provider
			email, _ := oauthSess.Values["email"].(string)
			name, _ := oauthSess.Values["name"].(string)

			user := models.User{}
			user.Name = name
			user.Email.SetValid(email)
			user.Provider.SetValid("google")
			user.ProviderID.SetValid(providerID)

			userID, err := dbAuthTemplate.FindOrCreateUserByProvider(debugTag, h.appConf.Db, user)
			if err != nil {
				log.Printf("%v failed user upsert: %v", debugTag, err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// 4) Create a DB session token for this user so rest of system can use existing auth model
			sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag, h.appConf.Db, userID, r.RemoteAddr, time.Time{})
			if err != nil {
				log.Printf("%v failed creating session token: %v", debugTag, err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// 5) Set the "session" cookie so subsequent requests use DB-backed auth
			http.SetCookie(w, sessionToken)

			// 6) Attach session to context and continue
			sess := &models.Session{UserID: userID}
			ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
