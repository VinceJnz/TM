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

type Session struct {
	Token   models.Token
	User    models.User
	Control models.Control
	//Message mdlMessage.PageMsg
}

type HandlerFunc func(http.ResponseWriter, *http.Request)

//type Handler struct {
//	appConf *appCore.Config
//}

type Handler struct {
	appConf *appCore.Config
	Pool    map[string]*poolItem
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
		//srvc:    app.Service,
		//app:     app,
		Pool: make(map[string]*poolItem),
	}
}

//func New(appConf *appCore.Config) *Handler {
//	return &Handler{appConf: appConf}
//}

func (h *Handler) RequireRestAuthTst(next http.Handler) http.Handler {
	session := &Session{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//var err error

		err := h.setRestResource(session, r)
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireRestAuth()1", "err =", err, "session =", session, "r =", r)
		}

		ctx := context.WithValue(r.Context(), h.appConf.UserIDKey, session.User.ID)
		next.ServeHTTP(w, r.WithContext(ctx)) // Access is correct so the request is passed to the next handler

	})
}

//RequireUserAuth The input to this is a function of the form "fn(Session, ResponseWriter, Request)"
//The return from this function is "http.HandlerFunc"
//This function uses an anonymouse function to form a closure around "var Session"
// ??????? The CheckAccess function may need to be rewritten so that it can be called by each handler
// ??????? or possibly be added as a seperate wrapper?????

// RequireRestAuth checks that the request is authorised, i.e. the user has been given a cookie by loging on.
// func (h *Handler) RequireRestAuth(fn func(http.ResponseWriter, *http.Request, *mdlSession.Item)) http.HandlerFunc {
func (h *Handler) RequireRestAuth(fn HandlerFunc) http.HandlerFunc {
	session := &Session{}
	//log.Println(debugTag + "Handler.RequireRestAuth()1")

	//anonymous function. This is returned by this function and called via Mux.HandleFunc
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var accessTypeID int

		err = h.setRestResource(session, r)
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"Handler.RequireRestAuth()2", "err =", err, "session =", session, "r =", r)
		}
		sessionToken, err := r.Cookie("session")
		if err == http.ErrNoCookie { // If there is no session cookie
			log.Printf("%v %v %v %v %+v %v %+v %v %+v %v %+v\n", debugTag+"Handler.RequireRestAuth()3", "err =", err, "session.Token =", session.Token, "session.User =", session.User, "session.Control =", session.Control, "r =", r)
			w.WriteHeader(http.StatusNetworkAuthenticationRequired)
			w.Write([]byte("Logon required - You don't have access to the requested resource."))
			return
		} else { // If there is a session cookie try to find it in the repository
			//_, err = h.srvc.CheckToken(session, sessionToken.Value)
			session.Token, err = h.FindSessionToken(sessionToken.Value)
			session.User.ID = int(session.Token.UserID)
			if err != nil { // could not find user sessionToken so user is not authorised
				log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"Handler.RequireRestAuth()4", "err =", err, "session =", session, "r =", r)
				//w.WriteHeader(http.StatusUnauthorized)
				//w.WriteHeader(http.StatusForbidden)
				w.WriteHeader(http.StatusNetworkAuthenticationRequired)
				w.Write([]byte("Token not authorised - You don't have access to the requested resource."))
				return
			}
		}
		accessTypeID, err = h.UserCheckAccess(session.User.ID, session.Control.ResourceName, session.Control.AccessLevel)
		if err != nil { // user doesn't have correct access to the resource
			log.Printf("%v %v %v %v %+v %v %+v %v %+v\n", debugTag+"Handler.RequireRestAuth()5", "err =", err, "session =", session, "session.User =", session.User, "r =", r)
			http.Error(w, "You don't have access to the requested resource.", http.StatusForbidden)
			return
		}
		session.Control.AccessTypeID = accessTypeID
		fn(w, r) // Access is correct so the request is passed to the next handler
	}
}

// SetRestResource Splits the request url and extracts the resource being accessed and what level of access is being requested
// This is used to determine if a user is permitted to access the resource
// func setRestResource(session *mdlSession.Session, r *http.Request) error {
func (h *Handler) setRestResource(session *Session, r *http.Request) error {
	var err error
	var urlSplit []string
	var apiVersion string

	session.Control.PrevURL = r.URL.Path //PrevURL is written to some of the forms in the browser so that it can be supplied back to the server when a form is submitted
	urlSplit = strings.Split(session.Control.PrevURL, "/")
	err = errors.New(debugTag + "SetRestResource()2 - Resource info not found") //this is the error returned if a valid resource is not identified
	session.Token.Host.SetValid(r.RemoteAddr)
	if urlSplit[1] == "api" {
		apiVersion = urlSplit[2]
		switch {
		case apiVersion == "v1":
			switch {
			case len(urlSplit) == 3:
				session.Control.ResourceName = urlSplit[3]
				session.Control.AccessLevel = r.Method //???? get, put, del, ...
				err = nil
				//log.Println(debugTag+"SetRestResource()2 ", "r.Method =", r.Method, "session =", session, "urlSplit =", urlSplit, "len(urlSplit) =", len(urlSplit), "session.Control.ResourceName =", session.Control.ResourceName, "session.Control.AccessLevel =", session.Control.AccessLevel, "err =", err)
			case len(urlSplit) == 4:
				session.Control.ResourceName = urlSplit[3]
				session.Control.AccessLevel = r.Method //???? get, put, del, ...
				err = nil
				//log.Println(debugTag+"SetRestResource()3 ", "r.Method =", r.Method, "session =", session, "urlSplit =", urlSplit, "len(urlSplit) =", len(urlSplit), "session.Control.ResourceName =", session.Control.ResourceName, "session.Control.AccessLevel =", session.Control.AccessLevel, "err =", err)
			case len(urlSplit) == 5:
				session.Control.ResourceName = urlSplit[3]
				session.Control.AccessLevel = r.Method //???? get, put, del, ...
				err = nil
				//log.Println(debugTag+"SetRestResource()4 ", "r.Method =", r.Method, "session =", session, "urlSplit =", urlSplit, "len(urlSplit) =", len(urlSplit), "session.Control.ResourceName =", session.Control.ResourceName, "session.Control.AccessLevel =", session.Control.AccessLevel, "err =", err)
			case len(urlSplit) == 6:
				session.Control.ResourceName = urlSplit[3]
				session.Control.AccessLevel = r.Method //???? get, put, del, ...
				err = nil
				//log.Println(debugTag+"SetRestResource()5 ", "r.Method =", r.Method, "session =", session, "urlSplit =", urlSplit, "len(urlSplit) =", len(urlSplit), "session.Control.ResourceName =", session.Control.ResourceName, "session.Control.AccessLevel =", session.Control.AccessLevel, "err =", err)
			case len(urlSplit) == 7:
				session.Control.ResourceName = urlSplit[5]
				session.Control.AccessLevel = r.Method //???? get, put, del, ...
				err = nil
				//log.Println(debugTag+"SetRestResource()6 ", "r.Method =", r.Method, "session =", session, "urlSplit =", urlSplit, "len(urlSplit) =", len(urlSplit), "session.Control.ResourceName =", session.Control.ResourceName, "session.Control.AccessLevel =", session.Control.AccessLevel, "err =", err)
			}
		}
	}
	if err != nil {
		log.Println(debugTag+"SetRestResource()7 ", "session =", session, "err =", err, "urlSplit =", urlSplit, "len(urlSplit) =", len(urlSplit), "session.Control.ResourceName =", session.Control.ResourceName, "session.Control.AccessLevel =", session.Control.AccessLevel)
		log.Printf("%v %v %v %v %v %+v", debugTag+"SetRestResource()8", "urlSplit =", urlSplit, len(urlSplit), "r =", r)
	}
	return err
}
