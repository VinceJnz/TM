package handlerAuth

import (
	"api-server/v2/dbTemplates/dbAuthTemplate"
	"api-server/v2/models"
	"log"
	"net/http"
)

func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(models.Session) // Used to retrieve the userID from the context so that access level can be assessed.
	// Need to check that the user is authorised to logout from the session provided (prevents anyone loging out anyone????
	//log.Printf(debugTag+"Handler.AuthLogout()1 userID=%v", session.UserID)

	if ok {
		sessionToken, err := r.Cookie("session")
		if err != nil {
			log.Printf(debugTag+"AuthLogout()2 session not open. userID=%v\n", session.UserID)
			http.Error(w, "session not open", http.StatusInternalServerError)
			return
		}
		h.removeSessionToken(sessionToken.Value)
	} else {
		log.Printf(debugTag+"AuthLogout()3 UserID not available in request context. userID=%v\n", session.UserID)
		http.Error(w, "UserID not available in request context", http.StatusInternalServerError)
		return
	}
	//w.WriteHeader(http.StatusNotImplemented)
	//w.Write([]byte("Not implemented"))

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("User logged out."))
}

func (h *Handler) removeSessionToken(tokenStr string) error {
	//tokenItem, err := h.FindSessionToken(tokenStr)
	tokenItem, err := dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, tokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken()1 token not found", "err =", err, "tokenItem =", tokenItem)
		return err
	}
	//err = h.TokenDeleteQry(tokenItem.ID)
	err = dbAuthTemplate.TokenDeleteQry(debugTag+"Handler.removeSessionToken()2 ", h.appConf.Db, tokenItem.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken()2 failed to remove token", "err =", err, "tokenItem =", tokenItem)
		return err
	}
	return nil
}
