package handlerAuth

import (
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
)

// SessionCheck is used by the client to see if the token it is using is still valid, if it is valid the the client is still logged in.
func (h *Handler) SessionCheck(w http.ResponseWriter, r *http.Request) {
	var err error
	var token models.Token
	var user models.User
	restResource, err := h.setRestResource(r) //stores prev-URL for redirect
	if err != nil {
		log.Println(debugTag+"Handler.SessionCheckRestHandler()1", "err =", err, "restResource =", restResource, "r =", r)
	}
	sessionToken, err := r.Cookie("session")
	if err == http.ErrNoCookie { // If there is no session cookie
		log.Println(debugTag+"Handler.SessionCheckRestHandler()2 - Authentication required ", "sessionToken=", sessionToken, "err =", err)
		w.WriteHeader(http.StatusNetworkAuthenticationRequired)
		w.Write([]byte("Logon required - You don't have access to the requested resource."))
		return
	} else { // If there is a session cookie try to find it in the repository
		token, err = h.FindSessionToken(sessionToken.Value) //This succeeds if the cookie is in the DB and the user is current
		//user.User.ID = user.Session.UserID
		if err != nil { // could not find user sessionToken so user is not authorised
			log.Println(debugTag+"Handler.SessionCheckRestHandler()3 - Not authorised ", "token =", token, "err =", err)
			w.WriteHeader(http.StatusNetworkAuthenticationRequired)
			w.Write([]byte("Token not authorised - You don't have access to the requested resource."))
			return
		} else { //Session cookie found, get user details and return to client
			user, err = h.UserReadQry(token.UserID)
			if err != nil {
				log.Printf("%v %v %v %v %+v %v %+v", debugTag+"Handler.SessionCheckRestHandler()8 - User not found", "err =", err, "user =", user, "sessionToken =", sessionToken)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("User not found."))
				return
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
