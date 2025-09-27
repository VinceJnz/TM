package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/srpPool"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

const debugTag = "handlerAuth."

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Handler struct {
	appConf *appCore.Config
	Pool    *srpPool.Pool
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
		Pool:    srpPool.NewSRPPool(),
	}
}

//RequireUserAuth The input to this is a function of the form "fn(Session, ResponseWriter, Request)"
//The return from this function is "http.HandlerFunc"
//This function uses an anonymouse function to form a closure around "var Session"
// ??????? The CheckAccess function may need to be rewritten so that it can be called by each handler
// ??????? or possibly be added as a seperate wrapper?????

// RequireRestAuth checks that the request is authorised, i.e. the user has been given a cookie by loging on.
func (h *Handler) RequireRestAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resource restResource
		var token models.Token
		var accessCheck models.AccessCheck
		var err error

		//token.Host.SetValid(r.RemoteAddr) // Do we need to check the host when we check the session cookie???

		sessionCookie, err := r.Cookie("session")
		if err == http.ErrNoCookie { // If there is no session cookie
			log.Printf("%v %v %v %v %v %v %+v\n", debugTag+"Handler.RequireRestAuth()1", "err =", err, "sessionCookie =", sessionCookie, "r =", r)
			w.WriteHeader(http.StatusNetworkAuthenticationRequired)
			w.Write([]byte("Logon required."))
			return
		} else { // If there is a session cookie try to find it in the repository
			//token, err = h.FindSessionToken(sessionCookie.Value)
			token, err = dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sessionCookie.Value)
			if err != nil { // could not find user session cookie in DB so user is not authorised
				log.Printf("%v %v %v %v %v %v %+v %v %+v\n", debugTag+"Handler.RequireRestAuth()2", "err =", err, "sessionCookie =", sessionCookie, "token =", token, "r =", r)
				w.WriteHeader(http.StatusNetworkAuthenticationRequired)
				w.Write([]byte("Logon required."))
				return
			} else {
				resource, err = h.setRestResource(r)
				if err != nil {
					log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireRestAuth()3", "err =", err, "resource =", resource, "r =", r)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Not authorised - You don't have access to the requested resource."))
					return
				} else {
					// check access to resource
					//accessCheck, err = h.UserCheckAccess(token.UserID, resource.ResourceName, resource.AccessMethod)
					accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireRestAuth()3a ", h.appConf.Db, token.UserID, resource.ResourceName, resource.AccessMethod)
					if err != nil {
						log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireRestAuth()4", "err =", err, "resource =", resource, "r =", r)
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte("Not authorised - You don't have access to the requested resource."))
						return
					}
				}
			}
		}

		session := &models.Session{
			UserID:         token.UserID,
			PrevURL:        resource.PrevURL,
			ResourceName:   resource.ResourceName,
			ResourceID:     0,
			AccessMethod:   resource.AccessMethod,
			AccessMethodID: 0,
			AccessType:     "",
			AccessTypeID:   accessCheck.AccessTypeID,
			AdminFlag:      accessCheck.AdminFlag,
		}

		log.Printf("%v %v %v %v %v %v %v %v %v %v %v %v %v\n", debugTag+"Handler.RequireRestAuth()5", "UserID =", session.UserID, "PrevURL =", session.PrevURL, "ResourceName =", session.ResourceName, "AccessMethod =", session.AccessMethod, "AccessType =", session.AccessType, "AdminFlag =", session.AdminFlag)
		//w.WriteHeader(http.StatusOK) // If this get called first, subsequent calls to w.WriteHeader are ignored. So it should not be called here.
		ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, session) // Store userID in the context. This can be used to filter rows in subsequent handlers
		next.ServeHTTP(w, r.WithContext(ctx))                                  // Access is correct so the request is passed to the next handler
	})
}

type restResource struct {
	PrevURL      string
	AccessMethod string
	ResourceName string
}

// SetRestResource Splits the request url and extracts the resource being accessed and what level of access is being requested
// This is used to determine if a user is permitted to access the resource
// func setRestResource(session *mdlSession.Session, r *http.Request) error {
func (h *Handler) setRestResource(r *http.Request) (restResource, error) {
	var err error
	var urlSplit []string
	var apiVersion string
	var control restResource

	control.PrevURL = r.URL.Path //PrevURL is written to some of the forms in the browser so that it can be supplied back to the server when a form is submitted
	urlSplit = strings.Split(control.PrevURL, "/")
	if len(urlSplit) == 0 {
		log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()2 ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
		err = errors.New(debugTag + "SetRestResource()1 - Resource info not found") //this is the error returned if a valid resource is not identified
		return restResource{}, err
	}
	control.AccessMethod = r.Method // get, put, post, del, ...
	switch urlSplit[1] {
	case "api":
		apiVersion = urlSplit[2]
		switch apiVersion {
		case "v1":
			switch len(urlSplit) {
			case 3:
				control.ResourceName = urlSplit[3]
			case 4:
				control.ResourceName = urlSplit[3]
			case 5:
				control.ResourceName = urlSplit[3]
			case 6:
				control.ResourceName = urlSplit[3]
			case 7:
				control.ResourceName = urlSplit[5]
			default:
				log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()4 invalid url: ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
				return restResource{}, errors.New(debugTag + "setRestResource()4 invalid url")
			}
		default:
			log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()5 invalid url: ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
			return restResource{}, errors.New(debugTag + "setRestResource()5 invalid url")
		}
	default:
		log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()6 invalid url: ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
		return restResource{}, errors.New(debugTag + "setRestResource()6 invalid url")
	}
	return control, err
}

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
		//token, err = h.FindSessionToken(sessionToken.Value) //This succeeds if the cookie is in the DB and the user is current
		token, err = dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sessionToken.Value)
		//user.User.ID = user.Session.UserID
		if err != nil { // could not find user sessionToken so user is not authorised
			log.Println(debugTag+"Handler.SessionCheckRestHandler()3 - Not authorised ", "token =", token, "err =", err)
			w.WriteHeader(http.StatusNetworkAuthenticationRequired)
			w.Write([]byte("Token not authorised - You don't have access to the requested resource."))
			return
		} else { //Session cookie found, get user details and return to client
			//user, err = h.UserReadQry(token.UserID)
			user, err := dbAuthTemplate.UserReadQry(debugTag+"Handler.AccountValidate()7a ", h.appConf.Db, token.UserID)
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
