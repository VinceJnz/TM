package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/models"
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
)

const debugTag = "handlerAuth."

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Handler struct {
	appConf *appCore.Config
	Pool    poolList
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
		//srvc:    app.Service,
		//app:     app,
		Pool: poolList{},
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
			token, err = h.FindSessionToken(sessionCookie.Value)
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
					accessCheck, err = h.UserCheckAccess(token.UserID, resource.ResourceName, resource.AccessMethod)
					if err != nil {
						log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireRestAuth()4", "err =", err, "resource =", resource, "r =", r)
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte("Not authorised - You don't have access to the requested resource."))
						return
					}
				}
			}
		}

		session := models.Session{
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
