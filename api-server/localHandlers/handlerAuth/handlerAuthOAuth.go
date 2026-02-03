package handlerAuth

import (
	"context"
	"log"
	"net/http"
	"time"

	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
)

// RequireOAuthOrSessionAuth returns a middleware that accepts either a DB session cookie ("session")
// or an OAuth session ("auth-session") from the provided gateway. It ensures an internal session
// context (models.Session) is available for downstream handlers.

func (h *Handler) RequireOAuthOrSessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resource RestResource
		var token models.Token
		var accessCheck models.AccessCheck
		var user models.User
		var err error

		// 1) Try existing DB session cookie
		if sc, err := r.Cookie("session"); err == nil {
			if token, err := dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sc.Value); err == nil {
				// build session context (same as RequireRestAuth)

				resource, err = h.setRestResource(r)
				if err != nil {
					log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuth()1", "err =", err, "resource =", resource, "r =", r)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Not authorised - You don't have access to the requested resource."))
					return
				} else {
					// check access to resource
					//accessCheck, err = h.UserCheckAccess(token.UserID, resource.ResourceName, resource.AccessMethod)
					accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireOAuth()1a ", h.appConf.Db, token.UserID, resource.ResourceName, resource.AccessMethod)
					if err != nil {
						log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuth()2", "err =", err, "resource =", resource, "r =", r)
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte("Not authorised - You don't have access to the requested resource."))
						return
					}
				}
				user, err = dbAuthTemplate.UserReadQry(debugTag+"", h.appConf.Db, token.UserID)
				if err != nil {
					log.Printf("%v failed reading user record: %v", debugTag, err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				// 6) Attach session to context and continue
				session := &models.Session{
					UserID:         token.UserID,
					PrevURL:        resource.PrevURL,
					ResourceName:   resource.ResourceName,
					ResourceID:     0,
					AccessMethod:   resource.AccessMethod,
					AccessMethodID: 0,
					AccessType:     "",
					AccessTypeID:   accessCheck.AccessTypeID,
					AdminFlag:      accessCheck.AdminFlag,
					Email:          user.Email.String,
				}

				ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, session)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// If DB lookup failed, continue to try OAuth session
		}

		// 2) Try OAuth session (gorilla/sessions store)
		oauthSess, _ := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
		providerID, _ := oauthSess.Values["user_id"].(string)
		if providerID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// 3) Find or create an internal user record mapped to this provider
		email, _ := oauthSess.Values["email"].(string)
		name, _ := oauthSess.Values["name"].(string)

		user = models.User{}
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

		resource, err = h.setRestResource(r)
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuth()3", "err =", err, "resource =", resource, "r =", r)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Not authorised - You don't have access to the requested resource."))
			return
		} else {
			// check access to resource
			//accessCheck, err = h.UserCheckAccess(token.UserID, resource.ResourceName, resource.AccessMethod)
			accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireOAuth()3a ", h.appConf.Db, token.UserID, resource.ResourceName, resource.AccessMethod)
			if err != nil {
				log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuth()4", "err =", err, "resource =", resource, "r =", r)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Not authorised - You don't have access to the requested resource."))
				return
			}
		}

		// 6) Attach session to context and continue
		session := &models.Session{
			UserID:         userID,
			PrevURL:        resource.PrevURL,
			ResourceName:   resource.ResourceName,
			ResourceID:     0,
			AccessMethod:   resource.AccessMethod,
			AccessMethodID: 0,
			AccessType:     "",
			AccessTypeID:   accessCheck.AccessTypeID,
			AdminFlag:      accessCheck.AdminFlag,
			Email:          email,
		}

		//sess := &models.Session{UserID: userID}
		ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
