package helpers

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type MailSender interface {
	SendMail(to string, title string, message string) (bool, error)
}

var (
	ErrAuthTokenInvalid = errors.New("auth token invalid")
	ErrAuthUserNotFound = errors.New("auth user not found")
	ErrAuthUserInactive = errors.New("auth user inactive")
)

func FindUserByUsernameOrEmail(debugStr string, db *sqlx.DB, username, email string) (models.User, error) {
	if username != "" {
		return dbAuthTemplate.UserNameReadQry(debugStr+"FindUserByUsernameOrEmail:byName ", db, username)
	}
	if email != "" {
		return dbAuthTemplate.UserEmailReadQry(debugStr+"FindUserByUsernameOrEmail:byEmail ", db, email)
	}
	return models.User{}, errors.New("username or email required")
}

func CreateAndSetSessionCookie(debugStr string, w http.ResponseWriter, db *sqlx.DB, userID int, host string, expiration time.Time) error {
	sessionToken, err := dbAuthTemplate.CreateSessionToken(debugStr+"CreateAndSetSessionCookie", db, userID, host, expiration)
	if err != nil {
		log.Printf("%vCreateAndSetSessionCookie() failed to create session token: %v", debugStr, err)
		return err
	}
	http.SetCookie(w, sessionToken)
	return nil
}

func CreateNamedTokenAndSendEmail(
	debugStr string,
	db *sqlx.DB,
	emailSvc MailSender,
	userID int,
	host string,
	tokenName string,
	expiration time.Time,
	recipient string,
	subject string,
	buildBody func(token string) string,
	deleteOnSendFailure bool,
) (string, bool, error) {
	tokenCookie, err := dbAuthTemplate.CreateNamedToken(debugStr+"CreateNamedTokenAndSendEmail", db, true, userID, host, tokenName, expiration)
	if err != nil {
		return "", false, err
	}

	tokenValue := tokenCookie.Value
	body := ""
	if buildBody != nil {
		body = buildBody(tokenValue)
	}

	if emailSvc == nil {
		return tokenValue, false, nil
	}

	if _, err := emailSvc.SendMail(recipient, subject, body); err != nil {
		if deleteOnSendFailure {
			if tok, findErr := dbAuthTemplate.FindToken(debugStr+"CreateNamedTokenAndSendEmail:findTokenForDeletion", db, tokenName, tokenValue); findErr == nil {
				_ = dbAuthTemplate.TokenDeleteQry(debugStr+"CreateNamedTokenAndSendEmail:delOnSendFailure", db, tok.ID)
			}
		}
		return tokenValue, false, err
	}

	return tokenValue, true, nil
}

func WriteAcceptedText(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(msg))
}

func WriteUnauthorizedText(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusUnauthorized)
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Printf("helpers.WriteUnauthorizedText() failed to write response: %v", err)
	}
}

func WriteForbidden(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusForbidden)
}

func WriteInternalServerError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusInternalServerError)
}

func WriteAcceptedJSON(w http.ResponseWriter, payload any) {
	WriteJSON(w, http.StatusAccepted, payload)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("helpers.WriteJSON() failed to encode response: %v", err)
	}
}

func DecodeJSONBody(r *http.Request, payload any) error {
	return json.NewDecoder(r.Body).Decode(payload)
}

func SessionExpiryFromRememberMe(rememberMe bool) time.Time {
	if rememberMe {
		return time.Now().Add(30 * 24 * time.Hour)
	}

	return time.Time{}
}

func ResolveUserIDFromSessionOrOAuthState(debugStr string, r *http.Request, db *sqlx.DB) int {
	// Try 1: DB-backed session cookie
	if sc, err := r.Cookie("session"); err == nil {
		log.Printf("%vResolveUserIDFromSessionOrOAuthState() found session cookie", debugStr)
		if dbToken, tokenErr := dbAuthTemplate.FindSessionToken(debugStr+"ResolveUserIDFromSessionOrOAuthState:session", db, sc.Value); tokenErr == nil {
			log.Printf("%vResolveUserIDFromSessionOrOAuthState() using session for user %d", debugStr, dbToken.UserID)
			return dbToken.UserID
		}
	}

	// Try 2: oauth-state token (popup fallback)
	if c, err := r.Cookie("oauth-state"); err == nil {
		log.Printf("%vResolveUserIDFromSessionOrOAuthState() found oauth-state cookie", debugStr)
		if dbToken, tokenErr := dbAuthTemplate.FindToken(debugStr+"ResolveUserIDFromSessionOrOAuthState:oauth-state", db, "oauth-state", c.Value); tokenErr == nil {
			log.Printf("%vResolveUserIDFromSessionOrOAuthState() using oauth-state for user %d", debugStr, dbToken.UserID)
			return dbToken.UserID
		}
	}

	return 0
}

func VerifyOTPTokenAndLoadUser(debugStr string, db *sqlx.DB, tokenName, tokenValue string, requireActive bool) (int, models.User, error) {
	tok, err := dbAuthTemplate.FindToken(debugStr+"VerifyOTPTokenAndLoadUser:find", db, tokenName, tokenValue)
	if err != nil {
		return 0, models.User{}, fmt.Errorf("%w: %v", ErrAuthTokenInvalid, err)
	}

	userID := tok.UserID
	if userID == 0 {
		return 0, models.User{}, ErrAuthTokenInvalid
	}

	user, err := dbAuthTemplate.UserReadQry(debugStr+"VerifyOTPTokenAndLoadUser:read", db, userID)
	if err != nil {
		return 0, models.User{}, fmt.Errorf("%w: %v", ErrAuthUserNotFound, err)
	}

	if requireActive && !user.UserActive() {
		return 0, models.User{}, ErrAuthUserInactive
	}

	_ = dbAuthTemplate.TokenDeleteQry(debugStr+"VerifyOTPTokenAndLoadUser:del", db, tok.ID)

	return userID, user, nil
}

// NotifyAdminsUserReviewRequired sends review/activation notice to all admin users.
// It is best-effort and returns an error only when admin recipients cannot be loaded.
func NotifyAdminsUserReviewRequired(debugStr string, db *sqlx.DB, emailSvc MailSender, groupName string, userID int, username, email, name string) error {
	if emailSvc == nil {
		log.Printf("%vNotifyAdminsUserReviewRequired email service not configured", debugStr)
		return nil
	}

	groupName = strings.TrimSpace(groupName)
	candidateGroups := []string{}
	if groupName != "" {
		for _, g := range strings.Split(groupName, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				candidateGroups = append(candidateGroups, g)
			}
		}
	} else {
		candidateGroups = []string{"admin", "administrator", "admins"}
	}

	var adminEmails []string
	resolvedGroup := ""
	for _, g := range candidateGroups {
		emails, err := dbAuthTemplate.UserEmailsByRole(debugStr+"NotifyAdminsUserReviewRequired:adminEmails", db, g)
		if err != nil {
			continue
		}
		if len(emails) > 0 {
			adminEmails = emails
			resolvedGroup = g
			break
		}
	}
	if len(adminEmails) == 0 {
		log.Printf("%vNotifyAdminsUserReviewRequired no users found in any notification group candidates: %v", debugStr, candidateGroups)
		return nil
	}

	subject := "New account pending admin approval"
	body := fmt.Sprintf(
		"A new user account is pending admin approval.\n\nUser ID: %d\nUsername: %s\nEmail: %s\nName: %s\n\nPlease review and activate the account if appropriate.",
		userID,
		username,
		email,
		name,
	)

	for _, adminEmail := range adminEmails {
		adminEmail = strings.TrimSpace(adminEmail)
		if adminEmail == "" {
			continue
		}

		if _, sendErr := emailSvc.SendMail(adminEmail, subject, body); sendErr != nil {
			log.Printf("%vNotifyAdminsUserReviewRequired failed to notify admin %s for user %d: %v", debugStr, adminEmail, userID, sendErr)
		} else {
			log.Printf("%vNotifyAdminsUserReviewRequired recipient %s notified for user %d via group %q", debugStr, adminEmail, userID, resolvedGroup)
		}
	}

	return nil
}
