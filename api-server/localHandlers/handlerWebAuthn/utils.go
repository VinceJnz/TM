package handlerWebAuthn

import (
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
)

// createTemporaryToken and return *http.Token
func createTemporaryToken(host string) (*http.Cookie, error) {
	var err error
	expiration := time.Now().Add(3 * time.Minute) // Token valid for 3 minutes
	tempSessionToken := &http.Cookie{
		Name:    "session",
		Value:   uuid.NewV4().String(),
		Path:    "/",
		Domain:  host,
		Expires: expiration,
		Secure:  true, // Always true for HTTPS
		//RawExpires: "",
		//MaxAge:     0,
		HttpOnly: false, //https --> true, http --> false
		SameSite: http.SameSiteNoneMode,
		//SameSite: http.SameSiteLaxMode,
		//SameSite: http.SameSiteStrictMode,
		//Raw:        "",
		//Unparsed:   []string{},
	}
	return tempSessionToken, err
}
