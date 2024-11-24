package handlerAuth

import (
	"api-server/v2/models"
	"log"
	"net/http"
)

func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(models.Session) // Used to retrieve the userID from the context so that access level can be assessed.
	// Need to check that the user is authorised to logout from the session provided (prevents anyone loging out anyone????
	log.Printf("%v %v %+v", debugTag+"Handler.AuthLogout()1 ", "userID =", session.UserID)

	if ok {
		sessionToken, err := r.Cookie("session")
		if err != nil {
			return
		}
		h.removeSessionToken(sessionToken.Value)
	}
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Not implemented"))
}

func (h *Handler) removeSessionToken(tokenStr string) error {
	tokenItem, err := h.FindSessionToken(tokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken()1 token not found", "err =", err, "tokenItem =", tokenItem)
		return err
	}
	err = h.TokenDeleteQry(tokenItem.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken()2 failed to remove token", "err =", err, "tokenItem =", tokenItem)
		return err
	}
	return nil
}
