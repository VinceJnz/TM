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
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/mux"
)

// API Request/Response structures - used to decode JSON requests and encode JSON responses
type ResetRequest struct {
	//Username string `json:"username"`
	Email string `json:"email"`
}

type ResetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Generate cryptographically secure token
func generateSecureToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// Send reset email (replace with your email service)
func sendResetEmail(email, resetToken string) error {
	resetLink := fmt.Sprintf("https://localhost:8086/api/v1/webauthn/emailReset/finish/%s", resetToken)

	// In production, use your email service (SendGrid, AWS SES, etc.)
	log.Printf("📧 WebAuthn Reset Email sent to: %s", email)
	log.Printf("🔗 Reset Link: %s", resetLink)
	log.Printf("📝 Email Body:")
	log.Printf(`
Subject: Reset Your WebAuthn Credentials

Hi there,

You requested to reset your WebAuthn credentials (authenticators/passkeys).

Click this link to reset your credentials:
%s

This link expires in 1 hour for security.

If you didn't request this, please ignore this email.

Thanks!
`, resetLink)

	return nil
}

// Step 1: User requests reset via email
// BeginEmailResetHandler handles the initial request to reset WebAuthn credentials via email
// It checks if the user exists, has WebAuthn enabled, and sends a reset link
func (h *Handler) BeginEmailResetHandler(w http.ResponseWriter, r *http.Request) {
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
	user, err := dbAuthTemplate.UserEmailReadQry(debugTag+"Handler.BeginLogin()1a ", h.appConf.Db, req.Email)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginLogin()1: User not found", "err =", err, "userID =", user.ID, "r.RemoteAddr =", r.RemoteAddr)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if the user is active
	if user.AccountStatusID.Int64 != int64(models.AccountActive) { // Assuming 1 is the ID for an active user
		log.Printf("%sHandler.BeginLogin()2 Error: user = %+v, AccountStatusID = %v, AccountActive value= %v", debugTag, user, user.AccountStatusID.Int64, int64(models.AccountActive))
		http.Error(w, "User is not active", http.StatusForbidden)
		return
	}

	log.Printf("%sHandler.BeginLogin()3 User found: %+v, ID = %d, Email = %s", debugTag, user, user.ID, user.Email.String)

	// Always return success to prevent email enumeration attacks
	response := ResetResponse{
		Success: true,
		Message: "If an account with that email exists and has WebAuthn enabled, a reset link has been sent.",
	}

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

		// Store reset token with 1-hour expiry
		dbAuthTemplate.TokenWriteQry(debugTag+"Handler.InitiateResetHandler()1 ", h.appConf.Db, newToken)

		// Send reset email
		if err := sendResetEmail(user.Email.String, resetToken); err != nil {
			log.Printf("Failed to send reset email to %s: %v", user.Email.String, err)
			// Don't reveal email sending failure to prevent enumeration
		} else {
			log.Printf("WebAuthn reset email sent to user %d (%+v)", user.ID, user.Email)
		}
	} else {
		log.Printf("User %d (%+v) does not have WebAuthn enabled or no credentials found. No email sent.", user.ID, user)
	}

	log.Printf("WebAuthn reset request processed for user %d (%+v)", user.ID, user.Email)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Step 2: User clicks email link to complete reset
// FinishEmailResetHandler handles the final step of resetting WebAuthn credentials
// It validates the reset token, clears the user's WebAuthn credentials, and returns a success response
// It also deletes the token to prevent reuse
func (h *Handler) FinishEmailResetHandler(w http.ResponseWriter, r *http.Request) {
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
	resetToken, err := dbAuthTemplate.FindToken(debugTag+"Handler.AuthUpdate()7 ", h.appConf.Db, "webauthn_reset", token)
	if err != nil || resetToken.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid or expired reset token",
		})
		return
	}

	// Token is used. Delete it so that it can't be reused
	err = dbAuthTemplate.TokenDeleteQry(debugTag+"Handler.AuthUpdate()7 ", h.appConf.Db, resetToken.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthUpdate()7 ", "err =", err, "token =", resetToken)
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
	user, err := dbAuthTemplate.UserReadQry(debugTag+"Handler.AuthUpdate()8 ", h.appConf.Db, resetToken.UserID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User not found",
		})
		return
	}

	// Perform the WebAuthn reset
	credentialCount := len(user.Credentials)
	user.Credentials = []webauthn.Credential{} // Clear all credentials
	//user.WebAuthnUserID = nil                  // Clear WebAuthn ID // Don't clear WebAuthnUserID, as it may be needed for future????

	// Log the successful reset
	log.Printf("WebAuthn credentials reset completed for user %d (%+v). %d credentials removed.",
		user.ID, user.Email, credentialCount)

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
