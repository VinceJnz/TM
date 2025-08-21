package handlerWebAuthn

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Reset steps
// 1. User requests reset via the client by entering their email address
// 2. Server checks the email is valid and sends an email with a token
// 3. User enters token via client UI and the client requests a list of registered devices
// 4. Server checks token and returns a list of the user's regestered devices
// 5. The client UI displays the devices registered to the user and the user chooses which device to reset
//     Having this step should ensure that the user does not end up wit a lot of old device registrations.
// 6. The server resets/re-registers the device and notifies the user ????

// 1. Request Reset
// API Request/Response structures - used to decode JSON requests and encode JSON responses
// POST /webauthn/reset/request
type ResetRequest struct {
	//Username string `json:"username"`
	Email string `json:"email"`
}

type ResetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// 2. Validate Token & List Devices
// POST /webauthn/reset/list
type ListDevicesRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}
type DeviceInfo struct {
	CredentialID string `json:"credential_id"`
	DeviceName   string `json:"device_name"`
	Created      string `json:"created"`
}
type ListDevicesResponse struct {
	Devices []DeviceInfo `json:"devices"`
}

// 3. Delete Credential
// POST /webauthn/reset/delete
type DeleteCredentialRequest struct {
	Email        string `json:"email"`
	Token        string `json:"token"`
	CredentialID string `json:"credential_id"`
}
type DeleteCredentialResponse struct {
	Success bool `json:"success"`
}

// Generate cryptographically secure token
func generateSecureToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// Send reset email (replace with your email service)
func (h *Handler) sendResetEmail(email, resetToken string) error {
	resetLink := fmt.Sprintf("https://localhost:8086/api/v1/webauthn/emailReset/finish/%s", resetToken)
	title := fmt.Sprintf("📧 WebAuthn Reset Email sent to: %s", email)
	message := fmt.Sprintf(`
Subject: Reset Your WebAuthn Credentials

Hi there,

You requested to reset your WebAuthn credentials (authenticators/passkeys).

Click this link to reset your credentials:
%s

Paste this token into the client to continue with the credential reset:
%s

This link expires in 1 hour for security.

If you didn't request this, please ignore this email.

Thanks!
`, resetLink, resetToken)

	// In production, use your email service (SendGrid, AWS SES, etc.)
	log.Printf("%ssendResetEmail()1 Send Reset Email", debugTag)
	log.Printf("%s", title)
	log.Printf("🔗 Reset Link: %s", resetLink)
	log.Printf("📝 Email Body:")
	log.Printf("%s", message)

	//h.appConf.EmailSvc.SendMail(email, title, message)
	h.appConf.EmailSvc.SendMail("vince.jennings@gmail.com", title, message) // ???? For debugging ????

	return nil
}

// Step 1: User Requests Reset (Server checks the email is valid and sends an email with a token)
// BeginEmailResetHandler handles the initial request to reset WebAuthn credentials via email
// It checks if the user exists, has WebAuthn enabled, and sends a reset link
func (h *Handler) EmailResetRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Get user email address from the request body
	var req ResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email is required"})
		return
	}

	// Get user email address from the request parameters
	//params := mux.Vars(r)
	//user_email := params["email"]
	//if user_email == "" {
	//	http.Error(w, "Invalid email", http.StatusBadRequest)
	//	return
	//}

	// Find user by email
	user, err := dbAuthTemplate.UserEmailReadQry(debugTag+"Handler.BeginEmailResetHandler()1a ", h.appConf.Db, req.Email)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginEmailResetHandler()1: User not found", "err =", err, "userID =", user.ID, "r.RemoteAddr =", r.RemoteAddr)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if the user is active
	if user.AccountStatusID.Int64 != int64(models.AccountActive) { // Assuming 1 is the ID for an active user
		log.Printf("%sHandler.BeginEmailResetHandler()2 Error: user = %+v, AccountStatusID = %v, AccountActive value= %v", debugTag, user, user.AccountStatusID.Int64, int64(models.AccountActive))
		http.Error(w, "User is not active", http.StatusForbidden)
		return
	}

	log.Printf("%sHandler.BeginEmailResetHandler()3 User found: %+v, ID = %d, Email = %s", debugTag, user, user.ID, user.Email.String)

	// Only send email if user exists and has WebAuthn credentials
	//if user.WebAuthnEnabled() && user.WebAuthnHasCredentials() {
	if user.WebAuthnEnabled() {
		// Generate secure reset token
		resetToken := generateSecureToken()

		newToken := models.Token{}
		newToken.UserID = user.ID
		newToken.TokenStr.SetValid(resetToken)
		newToken.Name.SetValid("webauthn_reset")
		newToken.Valid.SetValid(true)
		newToken.Host.SetValid(h.appConf.Settings.Host)
		newToken.ValidFrom.SetValid(time.Now())
		newToken.ValidTo.SetValid(time.Now().Add(1 * time.Hour)) // Token valid for 1 hour

		user.AccountStatusID.SetValid(int64(models.AccountWebAuthnResetResetRequired))
		// Store reset token with 1-hour expiry
		dbAuthTemplate.TokenWriteQry(debugTag+"Handler.BeginEmailResetHandler()4 ", h.appConf.Db, newToken)
		dbAuthTemplate.UserWriteQry(debugTag+"Handler.BeginEmailResetHandler()5 ", h.appConf.Db, user)

		// Send reset email
		log.Printf("%sHandler.BeginEmailResetHandler()5 user.Email = %+v", debugTag, user.Email.String)
		//if err := sendResetEmail(user.Email.String, resetToken); err != nil {
		if err := h.sendResetEmail(user.Email.String, resetToken); err != nil {
			log.Printf("Failed to send reset email to %s: %v", user.Email.String, err)
			// Don't reveal email sending failure to prevent enumeration
		} else {
			log.Printf("WebAuthn reset email sent to user %d (%+v)", user.ID, user.Email)
		}
	} else {
		log.Printf("User %d (%+v) does not have WebAuthn enabled or no credentials found. No email sent.", user.ID, user)
	}

	log.Printf("WebAuthn reset request processed for user %d (%+v)", user.ID, user.Email)

	// Always return success to prevent email enumeration attacks
	response := ResetResponse{
		Success: true,
		Message: "If an account with that email exists and has WebAuthn enabled, a reset link has been sent.",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Step 3: User Enters Token and Requests Device List (Server checks token and returns a list of the user's regestered devices)
func (h *Handler) EmailResetListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("%sHandler.FinishEmailResetHandler()1 Start", debugTag)

	params := mux.Vars(r)
	token := params["token"]
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Reset token is required",
		})
		return
	}

	// Validate reset token
	resetToken, err := dbAuthTemplate.FindToken(debugTag+"Handler.FinishEmailResetHandler()2 ", h.appConf.Db, "webauthn_reset", token)
	if err != nil || resetToken.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid or expired reset token",
		})
		return
	}

	// Token is used. Delete it so that it can't be reused
	err = dbAuthTemplate.TokenDeleteQry(debugTag+"Handler.FinishEmailResetHandler()3 ", h.appConf.Db, resetToken.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishEmailResetHandler()4 ", "err =", err, "token =", resetToken)
	}

	// Check if token is expired
	if time.Now().After(resetToken.ValidTo.Time) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Reset token has expired",
		})
		return
	}

	// Find the user
	user, err := dbAuthTemplate.UserReadQry(debugTag+"Handler.FinishEmailResetHandler()5 ", h.appConf.Db, resetToken.UserID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User not found",
		})
		return
	}

	log.Printf("WebAuthn user %d found (%+v). Token check successfully completed.", user.ID, user.Email)

	// Retrieve user credentials for each device and send it to the client so that they can select which device to reset
	credentials, err := dbAuthTemplate.GetUserCredentials(debugTag+"Handler.FinishEmailResetHandler()6 ", h.appConf.Db, user.ID)
	if err != nil {
		log.Printf(debugTag+"Handler.FinishEmailResetHandler()6: user = %+v", user)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Credentials not found",
		})
		return
	}

	// Create and Store user in your session pool store
	tempSessionToken, err := dbAuthTemplate.CreateTemporaryToken(WebAuthnSessionCookieName, h.appConf.Settings.Host)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()4: Failed to create session token", "err =", err, "WebAuthnSessionCookieName =", WebAuthnSessionCookieName, "host =", h.appConf.Settings.Host)
		return
	}
	h.Pool.Add(tempSessionToken.Value, &user, nil, 5*time.Minute) // Assuming you have a pool to manage session data

	// add the session token to the response
	http.SetCookie(w, tempSessionToken)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(credentials)

}

// Step 6: Server Deletes Selected Credential (The server deleted credential and notifies the user ????)
// FinishEmailResetHandler handles the final step of resetting WebAuthn credentials
// The user must then reregisster the device
func (h *Handler) EmailResetFinishHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("%sHandler.EmailResetFinishHandler()1 Start", debugTag)

	tempSessionToken, err := getTempSessionToken(w, r) // Retrieve the session cookie
	if err != nil {
		log.Println(debugTag+"Handler.EmailResetFinishHandler()2 - Authentication required ", "sessionToken=", tempSessionToken, "err =", err)
		return
	}
	poolItem, exists := h.Pool.Get(tempSessionToken.Value) // Assuming you have a pool to manage session data
	if !exists || poolItem.User == nil {                   // Check if the user is nil) {
		http.Error(w, "Session not found or expired", http.StatusUnauthorized)
		return
	}
	//sessionData := *poolItem.SessionData // Retrieve session data from the pool
	user := poolItem.User // Set the user data from the pool

	params := mux.Vars(r)
	credentialID := params["id"]
	if credentialID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Reset token is required"})
		return
	}

	// Token is used. Delete it so that it can't be reused
	credentialIDint, err := strconv.Atoi(credentialID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.EmailResetFinishHandler()3 ", "err =", err, "credentialID =", credentialID)

	}
	err = dbAuthTemplate.DeleteCredentialByID(debugTag+"Handler.EmailResetFinishHandler()4 ", h.appConf.Db, credentialIDint)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.EmailResetFinishHandler()5 ", "err =", err, "credentialIDint =", credentialIDint)
	}

	// Perform the WebAuthn reset
	credentialCount := len(user.Credentials)

	// Log the successful reset
	log.Printf("WebAuthn credentials reset completed for user %d (%+v). %d credentials removed.", user.ID, user.Email, credentialCount)

	// Return success response
	response := map[string]interface{}{
		"success":             true,
		"message":             "Your WebAuthn credentials have been successfully reset.",
		"credentials_removed": credentialCount,
		"next_steps": []string{
			"You can now log in using your password",
			"After logging in, re-register your WebAuthn authenticators for future passwordless login",
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Optional: Check token status
func (h *Handler) CheckTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token parameter required"})
		return
	}

	resetToken, err := dbAuthTemplate.FindToken(debugTag+"Handler.CheckTokenHandler()1 ", h.appConf.Db, "webauthn_reset", token)
	if err != nil || resetToken.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "Token not found",
		})
		return
	}

	// Mask email for privacy
	maskedEmail := func(email string) string {
		parts := strings.Split(email, "@")
		if len(parts) != 2 || len(parts[0]) <= 2 {
			return email
		}
		username := parts[0]
		domain := parts[1]
		masked := username[:1] + strings.Repeat("*", len(username)-2) + username[len(username)-1:]
		return masked + "@" + domain
	}

	status := map[string]interface{}{ // What does this do?????
		"valid":      time.Now().Before(resetToken.ValidTo.Time),
		"used":       true,
		"expired":    time.Now().After(resetToken.ValidTo.Time),
		"expires_at": resetToken.ValidTo.Time.Format(time.RFC3339),
		"email":      maskedEmail(""), // Need to get the user's email. Could add it to a poolItem or look it up by Token.UserID
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status) // Why do we return the status here?
}
