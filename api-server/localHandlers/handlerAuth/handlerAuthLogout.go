package handlerAuth

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"log"
	"net/http"
)

func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session) // Used to retrieve the userID from the context so that access level can be assessed.
	// Need to check that the user is authorised to logout from the session provided (prevents anyone loging out anyone????
	log.Printf(debugTag+"Handler.AuthLogout()1 session=%+v, ok=%v", session, ok)

	if ok {
		sessionToken, err := r.Cookie("session")
		if err != nil {
			log.Printf(debugTag+"AuthLogout()2 session not open. userID=%v\n", session.UserID)
			http.Error(w, "session not open", http.StatusInternalServerError)
			return
		}
		log.Printf(debugTag+"Handler.AuthLogout()2 session open. userID=%v, sessionToken=%+v\n", session.UserID, sessionToken)
		h.removeSessionToken(sessionToken.Value)
	} else {
		log.Printf(debugTag+"AuthLogout()3 UserID not available in request context. session=%+v\n", session)
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
	log.Printf(debugTag+"Handler.removeSessionToken()1 token found. tokenItem=%+v", tokenItem)
	//err = h.TokenDeleteQry(tokenItem.ID)
	err = dbAuthTemplate.TokenDeleteQry(debugTag+"Handler.removeSessionToken()2 ", h.appConf.Db, tokenItem.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken()2 failed to remove token", "err =", err, "tokenItem =", tokenItem)
		return err
	}
	err = dbAuthTemplate.UserDelProvider(debugTag+"Handler.removeSessionToken()3 ", h.appConf.Db, tokenItem.UserID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.removeSessionToken()3 failed to remove provider info from user record", "err =", err, "tokenItem =", tokenItem)
		return err
	}

	return nil
}
