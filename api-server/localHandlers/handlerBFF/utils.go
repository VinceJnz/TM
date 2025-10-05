package handlerBFF

import (
	"fmt"
	"log"
	"net/http"
)

// getTempSessionToken retrieves the temporary token from the request and processes errors.
func (h *Handler) getTempSessionToken(w http.ResponseWriter, r *http.Request) (*http.Cookie, error) {
	tempSessionToken, err := r.Cookie(BFFSessionTokenName) // Retrieve the session cookie
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

// sendRegistrationEmail sends a registration email to the user.
func (h *Handler) sendRegistrationEmail(email, token string) error {
	title := fmt.Sprintf("📧 BFF Registration Email sent to: %s", email)
	message := fmt.Sprintf(`
Subject: Register Your BFF Credentials

Hi there,

You requested to register your BFF credentials (authenticators/passkeys).

Paste this token into the client to continue with the credential registration:
%s

This link expires in 1 hour for security.

If you didn't request this, please ignore this email.

Thanks!
`, token)

	// In production, use your email service (SendGrid, AWS SES, etc.)
	log.Printf("%sSendRegistrationEmail()1 Send Registration Email", debugTag)
	log.Printf("%s", title)
	log.Printf("📝 Email Body:")
	log.Printf("%s", message)

	//h.appConf.EmailSvc.SendMail(email, title, message)
	h.appConf.EmailSvc.SendMail("vince.jennings@gmail.com", title, message) // ???? For debugging ????

	return nil
}

// sendRegistrationSMS sends a registration SMS to the user.
func (h *Handler) sendRegistrationSMS(phoneNumber, token string) error {
	return fmt.Errorf("SMS sending not implemented yet")
}
