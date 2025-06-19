package handlerWebAuthn

import (
	"fmt"
	"net/http"
)

// getSessionToken retrieves the session token from the request and processes errors.
func getSessionToken(w http.ResponseWriter, r *http.Request) (*http.Cookie, error) {
	tempSessionToken, err := r.Cookie(WebAuthnSessionCookieName) // Retrieve the session cookie
	if err != nil {
		switch err {
		case http.ErrNoCookie: // If there is no session cookie
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			//log.Println(debugTag+"Handler.FinishRegistration()1 - Authentication required ", "sessionToken=", tempSessionToken, "err =", err)
			var ErrAuthenticationRequired = fmt.Errorf("%sHandler.FinishRegistration()1 - Authentication required", debugTag)
			return nil, ErrAuthenticationRequired
		default: // If there is some other error
			http.Error(w, "Error retrieving session cookie", http.StatusInternalServerError)
			//log.Println(debugTag+"Handler.FinishRegistration()2 - Error retrieving session cookie ", "sessionToken=", tempSessionToken, "err =", err)
			var ErrSessionCookieRetrieval = fmt.Errorf("%sHandler.FinishRegistration()2 - Error retrieving session cookie", debugTag)
			return nil, ErrSessionCookieRetrieval
		}
	}
	return tempSessionToken, nil
}
