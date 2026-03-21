package handlerAuth

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"log"
	"net/http"
)

// AuthLogout ends the user's session by removing the session token.
// IMPORTANT: This does NOT disconnect OAuth providers - that requires a separate explicit action.
// OAuth provider linkage (provider, provider_id) remains intact for future logins.
// Users can log back in using the same OAuth provider without re-authorization.
func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session) // Used to retrieve the userID from the context so that access level can be assessed.
	log.Printf(debugTag+"Handler.AuthLogout session=%+v, ok=%v", session, ok)

	if ok {
		sessionToken, err := r.Cookie("session")
		if err != nil {
			log.Printf(debugTag+"AuthLogout session not open. userID=%v\n", session.UserID)
			http.Error(w, "session not open", http.StatusUnauthorized)
			return
		}
		log.Printf(debugTag+"Handler.AuthLogout session open. userID=%v, sessionToken=%+v\n", session.UserID, sessionToken)
		err = h.removeSessionToken(sessionToken.Value, session.UserID)
		if err != nil {
			log.Printf(debugTag+"Handler.AuthLogout failed to remove session token. userID=%v err=%v", session.UserID, err)
			http.Error(w, "failed to close session", http.StatusForbidden)
			return
		}
	} else {
		log.Printf(debugTag+"AuthLogout UserID not available in request context. session=%+v\n", session)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	if _, err := w.Write([]byte("User logged out.")); err != nil {
		log.Printf(debugTag+"AuthLogout failed to write response: %v", err)
	}
}

func (h *Handler) removeSessionToken(tokenStr string, expectedUserID int) error {
	tokenItem, err := dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, tokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken token not found", "err =", err, "tokenItem =", tokenItem)
		return err
	}
	if tokenItem.UserID != expectedUserID {
		log.Printf("%v token user mismatch: expected=%d got=%d", debugTag+"Handler.removeSessionToken", expectedUserID, tokenItem.UserID)
		return http.ErrNoCookie
	}
	log.Printf(debugTag+"Handler.removeSessionToken token found. tokenItem=%+v", tokenItem)
	err = dbAuthTemplate.TokenDeleteQry(debugTag+"Handler.removeSessionToken ", h.appConf.Db, tokenItem.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken failed to remove token", "err =", err, "tokenItem =", tokenItem)
		return err
	}
	return nil
}
