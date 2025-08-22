package handlerWebAuthn

import (
	"fmt"
	"log"
	"net/http"
)

// getTempSessionToken retrieves the temporary session token from the request and processes errors.
func (h *Handler) getTempSessionToken(w http.ResponseWriter, r *http.Request) (*http.Cookie, error) {
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

type SendMethod int

const (
	SendMethodEmail SendMethod = iota
	SendMethodSMS
)

func (h *Handler) sendTempSessionToken(address string, method SendMethod, tempToken string) {
	switch method {
	case SendMethodEmail:
		// Send the token via email
		h.sendRegistrationEmail(address, tempToken)
	case SendMethodSMS:
		// Send the token via SMS
		h.sendRegistrationSMS(address, tempToken)
	}
}

func (h *Handler) sendRegistrationEmail(email, token string) error {
	title := fmt.Sprintf("📧 WebAuthn Registration Email sent to: %s", email)
	message := fmt.Sprintf(`
Subject: Register Your WebAuthn Credentials

Hi there,

You requested to register your WebAuthn credentials (authenticators/passkeys).

Paste this token into the client to continue with the credential registration:
%s

This link expires in 1 hour for security.

If you didn't request this, please ignore this email.

Thanks!
`, token)

	// In production, use your email service (SendGrid, AWS SES, etc.)
	log.Printf("%sendRegistrationEmail()1 Send Registration Email", debugTag)
	log.Printf("%s", title)
	log.Printf("📝 Email Body:")
	log.Printf("%s", message)

	//h.appConf.EmailSvc.SendMail(email, title, message)
	h.appConf.EmailSvc.SendMail("vince.jennings@gmail.com", title, message) // ???? For debugging ????

	return nil
}

func (h *Handler) sendRegistrationSMS(email, token string) error {
	return fmt.Errorf("SMS sending not implemented yet")
}
