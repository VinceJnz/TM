package handlerAuth

import (
	"context"
	"log"
	"net/http"

	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
)

// The function RequireOAuthOrSessionAuth needs to do the following:
//1. Check to see the the users is already logged in (It already deos this)
//2. Try logging in using OAuth
//3. Try using a password and emailed token for authentication.

// RequireOAuthOrSessionAuth returns a middleware that accepts either a DB session cookie ("session")
// or an OAuth session ("auth-session") from the provided gateway. It ensures an internal session
// context (models.Session) is available for downstream handlers.
func (h *Handler) RequireOAuthOrSessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resource RestResource
		//var dbToken models.Token
		var accessCheck models.AccessCheck
		var user models.User
		//var err error
		log.Printf("%vRequireOAuthOrSessionAuth()0, checking auth for request: %v %v\n", debugTag, r.Method, r.URL.Path)

		// 1) Try existing DB session cookie to see if the user is already logged in
		if sc, err := r.Cookie("session"); err == nil {
			if dbToken, err := dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sc.Value); err == nil {
				// build session context (same as RequireRestAuth)

				//resource, err = h.setRestResource(r)
				if resource, err = h.setRestResource(r); err != nil {
					log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuthOrSessionAuth()1", "err =", err, "resource =", resource, "r =", r)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Not authorised - You don't have access to the requested resource."))
					return
				}
				// check access to resource
				//accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireOAuth()1a ", h.appConf.Db, token.UserID, resource.ResourceName, resource.AccessMethod)
				if accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireOAuthOrSessionAuth()1a ", h.appConf.Db, dbToken.UserID, resource.ResourceName, resource.AccessMethod); err != nil {
					log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuthOrSessionAuth()2", "err =", err, "resource =", resource, "r =", r)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Not authorised - You don't have access to the requested resource."))
					return
				}
				// Check the user record is still valid/active
				//user, err = dbAuthTemplate.UserReadQry(debugTag+"", h.appConf.Db, token.UserID)
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
					AdminFlag:      accessCheck.AdminFlag,
					Email:          user.Email.String,
				}
				// 7) Give the user access
				ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, session)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// If DB lookup failed, continue to try OAuth session
		}
		log.Printf("%vRequireOAuthOrSessionAuth()3 no valid DB session, trying OAuth session\n", debugTag)
		// 2) Try OAuth session (DB-backed token named "oauth-state")

		/*
				var providerID, email, name string

				// First try to find the provider ID in the OAuth session cookie set by the OAuth handlers
				if c, err := r.Cookie("oauth-state"); err == nil {
					if dbToken, err := dbAuthTemplate.FindToken(debugTag+"RequireOAuthOrSessionAuth:find_oauth_state", h.appConf.Db, "oauth-state", c.Value); err == nil {
							var sd map[string]any
							if dbToken.SessionData.Valid {
								_ = json.Unmarshal([]byte(dbToken.SessionData.String), &sd)
								if v, ok := sd["user_id"].(string); ok {
									providerID = v
								}
								if v, ok := sd["email"].(string); ok {
									email = v
								}
								if v, ok := sd["name"].(string); ok {
									name = v
								}
							}
					}
				}

				oauthSess, _ := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
				providerID, _ = oauthSess.Values["user_id"].(string)
				if providerID == "" {
					// OAuth session not present. Try password + emailed token fallback (Basic auth or token)
					// 3) Try password and emailed token for authentication.
					// 3a) Check Basic Auth credentials
					if username, password, ok := r.BasicAuth(); ok {
						// try to find user by username or email
						var foundUser models.User
						var uid int
						if u, err := dbAuthTemplate.UserNameReadQry(debugTag+"BasicAuthUser ", h.appConf.Db, username); err == nil {
							foundUser = u
							uid = u.ID
						} else if u, err := dbAuthTemplate.UserEmailReadQry(debugTag+"BasicAuthUserEmail ", h.appConf.Db, username); err == nil {
							foundUser = u
							uid = u.ID
						} else {
							// user not found
							http.Error(w, "unauthorized", http.StatusUnauthorized)
							return
						}
						// Instead of checking a stored password, require the user to submit
						// the emailed one-time token as the HTTP Basic password. Validate
						// the token and ensure it belongs to the user.
						tok, err := dbAuthTemplate.FindToken(debugTag+"BasicAuthEmailToken", h.appConf.Db, "_temp_email_token", password)
						if err != nil {
							log.Printf("%v Basic auth token not found or invalid: %v", debugTag, err)
							http.Error(w, "unauthorized", http.StatusUnauthorized)
							return
						}
						if tok.UserID != uid {
							log.Printf("%v Basic auth token user mismatch: tokenUser=%v submittedUser=%v", debugTag, tok.UserID, uid)
							http.Error(w, "unauthorized", http.StatusUnauthorized)
							return
						}
						// Optionally remove the one-time token so it cannot be reused
						if err := dbAuthTemplate.TokenDeleteQry(debugTag+"BasicAuthUseToken", h.appConf.Db, tok.ID); err != nil {
							log.Printf("%v failed to delete used token: %v", debugTag, err)
						}

						// Create DB session token for this user
						sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag, h.appConf.Db, uid, r.RemoteAddr, time.Time{})
						if err != nil {
							log.Printf("%v failed creating session token (basic auth): %v", debugTag, err)
							http.Error(w, "internal server error", http.StatusInternalServerError)
							return
						}
						http.SetCookie(w, sessionToken)

						// build resource/access context same as below
						resource, err := h.setRestResource(r)
						if err != nil {
							log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuthOrSessionAuth()3", "err =", err, "resource =", resource, "r =", r)
							w.WriteHeader(http.StatusUnauthorized)
							w.Write([]byte("Not authorised - You don't have access to the requested resource."))
							return
						}
						accessCheck, err := dbAuthTemplate.UserCheckAccess(debugTag+"RequireOAuthOrSessionAuth()3a ", h.appConf.Db, uid, resource.ResourceName, resource.AccessMethod)
						if err != nil {
							log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuthOrSessionAuth()4", "err =", err, "resource =", resource, "r =", r)
							w.WriteHeader(http.StatusUnauthorized)
							w.Write([]byte("Not authorised - You don't have access to the requested resource."))
							return
						}

						sess := &models.Session{
							UserID:         uid,
							PrevURL:        resource.PrevURL,
							ResourceName:   resource.ResourceName,
							ResourceID:     0,
							AccessMethod:   resource.AccessMethod,
							AccessMethodID: 0,
							AccessType:     "",
							AccessTypeID:   accessCheck.AccessTypeID,
							AdminFlag:      accessCheck.AdminFlag,
							Email:          foundUser.Email.String,
						}
						ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, sess)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}

					// 3b) Check for emailed token (header, cookie or query param)
					tokenStr := r.Header.Get("X-Email-Token")
					if tokenStr == "" {
						if c, err := r.Cookie("_temp_email_token"); err == nil {
							tokenStr = c.Value
						}
					}
					if tokenStr == "" {
						tokenStr = r.URL.Query().Get("token")
					}
					if tokenStr != "" {
						// find token record
						tok, err := dbAuthTemplate.FindToken(debugTag+"RequireOAuthToken", h.appConf.Db, "_temp_email_token", tokenStr)
						if err != nil {
							log.Printf("%v failed to find email token: %v", debugTag, err)
							http.Error(w, "unauthorized", http.StatusUnauthorized)
							return
						}
						// read user record
						userRec, err := dbAuthTemplate.UserReadQry(debugTag+"RequireOAuthTokenUser", h.appConf.Db, tok.UserID)
						if err != nil {
							log.Printf("%v failed to read user for token: %v", debugTag, err)
							http.Error(w, "internal server error", http.StatusInternalServerError)
							return
						}
						if !userRec.UserActive() {
							log.Printf("%v user account not active: %v", debugTag, userRec.ID)
							http.Error(w, "user account not active", http.StatusInternalServerError)
							return
						}

						// Create DB session token for this user
						sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag, h.appConf.Db, userRec.ID, r.RemoteAddr, time.Time{})
						if err != nil {
							log.Printf("%v failed creating session token (email token): %v", debugTag, err)
							http.Error(w, "internal server error", http.StatusInternalServerError)
							return
						}
						http.SetCookie(w, sessionToken)

						resource, err := h.setRestResource(r)
						if err != nil {
							log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuth()3", "err =", err, "resource =", resource, "r =", r)
							w.WriteHeader(http.StatusUnauthorized)
							w.Write([]byte("Not authorised - You don't have access to the requested resource."))
							return
						}
						accessCheck, err := dbAuthTemplate.UserCheckAccess(debugTag+"RequireOAuth()3a ", h.appConf.Db, userRec.ID, resource.ResourceName, resource.AccessMethod)
						if err != nil {
							log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireOAuth()4", "err =", err, "resource =", resource, "r =", r)
							w.WriteHeader(http.StatusUnauthorized)
							w.Write([]byte("Not authorised - You don't have access to the requested resource."))
							return
						}

						sess := &models.Session{
							UserID:         userRec.ID,
							PrevURL:        resource.PrevURL,
							ResourceName:   resource.ResourceName,
							ResourceID:     0,
							AccessMethod:   resource.AccessMethod,
							AccessMethodID: 0,
							AccessType:     "",
							AccessTypeID:   accessCheck.AccessTypeID,
							AdminFlag:      accessCheck.AdminFlag,
							Email:          userRec.Email.String,
						}
						ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, sess)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
					// No OAuth, no basic auth and no email token => unauthorised
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				// 3) Find or create an internal user record mapped to this provider
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
					accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireOAuth()3a ", h.appConf.Db, userID, resource.ResourceName, resource.AccessMethod)
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
		*/
	})
}
