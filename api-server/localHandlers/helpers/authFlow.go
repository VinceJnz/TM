package helpers

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
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

func CreateAndSetSessionCookie(ctx context.Context, debugStr string, w http.ResponseWriter, db *sqlx.DB, userID int, host string, expiration time.Time) error {
	sessionToken, err := dbAuthTemplate.CreateNamedTokenContext(ctx, debugStr+"CreateAndSetSessionCookie", db, true, userID, host, "session", expiration)
	if err != nil {
		log.Printf("%vCreateAndSetSessionCookie() failed to create session token: %v", debugStr, err)
		return err
	}
	http.SetCookie(w, sessionToken)
	return nil
}

func CreateNamedTokenAndSendEmail(
	ctx context.Context,
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
	tokenCookie, err := dbAuthTemplate.CreateNamedTokenContext(ctx, debugStr+"CreateNamedTokenAndSendEmail", db, true, userID, host, tokenName, expiration)
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

	var sendErr error
	if contextual, ok := emailSvc.(interface {
		SendMailContext(context.Context, string, string, string) (bool, error)
	}); ok {
		_, sendErr = contextual.SendMailContext(ctx, recipient, subject, body)
	} else {
		_, sendErr = emailSvc.SendMail(recipient, subject, body)
	}
	if err := sendErr; err != nil {
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("helpers.WriteJSON() failed to encode response: %v", err)
	}
}

func DecodeJSONBody(r *http.Request, payload any) error {
	const maxBody = 64 << 10
	if r.Body == nil {
		return errors.New("missing request body")
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return err
	}
	if len(data) > maxBody {
		return errors.New("request body too large")
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("request object required")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err = decoder.Decode(payload); err != nil {
		return err
	}
	var trailing any
	if err = decoder.Decode(&trailing); err != io.EOF {
		return errors.New("exactly one JSON value required")
	}
	return nil
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
		if dbToken, tokenErr := dbAuthTemplate.FindSessionTokenContext(r.Context(), debugStr+"ResolveUserIDFromSessionOrOAuthState:session", db, sc.Value); tokenErr == nil {
			log.Printf("%vResolveUserIDFromSessionOrOAuthState() using session for user %d", debugStr, dbToken.UserID)
			return dbToken.UserID
		}
	}

	return 0
}

func VerifyOTPTokenAndLoadUser(ctx context.Context, debugStr string, db *sqlx.DB, tokenName, tokenValue string, requireActive bool) (int, models.User, error) {
	tok, err := dbAuthTemplate.ConsumeToken(ctx, db, tokenName, tokenValue)
	if err != nil {
		return 0, models.User{}, fmt.Errorf("%w: %v", ErrAuthTokenInvalid, err)
	}

	userID := tok.UserID
	if userID == 0 {
		return 0, models.User{}, ErrAuthTokenInvalid
	}

	user, err := dbAuthTemplate.UserReadQryContext(ctx, debugStr+"VerifyOTPTokenAndLoadUser:read", db, userID)
	if err != nil {
		return 0, models.User{}, fmt.Errorf("%w: %v", ErrAuthUserNotFound, err)
	}

	if requireActive && !user.UserActive() {
		return 0, models.User{}, ErrAuthUserInactive
	}

	// Token was consumed atomically before account lookup.

	return userID, user, nil
}
