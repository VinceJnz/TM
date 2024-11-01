package handlerAuth

import (
	"api-server/v2/app"
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

type Handler struct {
	appConf *app.Config
}

func New(appConf *app.Config) *Handler {
	return &Handler{appConf: appConf}
}

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
		accessTypeID, err = h.CheckAccess(session.User.ID, session.Control.ResourceName, session.Control.AccessLevel)
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

const (
	//Finds only valid cookies where the user account is current
	//if the user account is disabled or set to new it will not return the cookie
	//if the cookie is not valid it will not return the cookie.
	sqlFindSessionToken = `SELECT c.ID, c.User_ID, c.Name, c.token, c.token_valid_ID, c.Valid_From, c.Valid_To
	FROM st_token c
		JOIN st_user u ON u.ID=c.User_ID
		LEFT JOIN se_token_valid sv ON sv.ID=c.token_valid_ID
	WHERE c.token=$1 AND c.Name='session' AND c.token_valid_ID=1 AND u.User_status_ID=1`

	//Finds valid tokens where user account exists and the token name is the same as the name passed in
	sqlFindToken = `SELECT c.ID, c.User_ID, c.Name, c.token, c.token_valid_ID, c.Valid_From, c.Valid_To
	FROM st_token c
		JOIN st_user u ON u.ID=c.User_ID
		LEFT JOIN se_token_valid sv ON sv.ID=c.token_valid_ID
	WHERE c.token=$1 AND c.Name=$2 AND c.token_valid_ID=1`
)

// FindSessionToken using the session cookie string find session cookie data in the DB and return the token item
// if the cookie is not found return the DB error
func (h *Handler) FindSessionToken(cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	//err = r.DBConn.QueryRow(sqlFindCookie, result.CookieStr).Scan(&result.ID, &result.UserID, &result.Name, &result.CookieStr, &result.Valid, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	err = h.appConf.Db.QueryRow(sqlFindSessionToken, result.TokenStr).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"SessionRepo.FindSessionToken()2", "err =", err, "sqlFindSessionToken =", sqlFindSessionToken, "result =", result)
		return result, err
	}
	return result, nil
}

// FindToken using the session cookie name and cookie string find session cookie data in the DB and return the token item
func (h *Handler) FindToken(name, cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	result.Name.SetValid(name)
	err = h.appConf.Db.QueryRow(sqlFindToken, result.TokenStr, result.Name).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"SessionRepo.FindToken()2", "err =", err, "sqlFindToken =", sqlFindToken, "result =", result)
		return result, err
	}
	return result, nil
}

// sqlCheckAccess checks that the user account has been activated and that it has access to the requested resource and method.
const (
	sqlCheckAccess = `SELECT eat.ID
	FROM st_user su
		JOIN st_user_group sug ON sug.User_ID=su.ID
		JOIN st_group sg ON sg.ID=sug.Group_ID
		JOIN st_group_resource sgr ON sgr.Group_ID=sg.ID
		JOIN se_resource er ON er.ID=sgr.Resource_ID
		JOIN se_access_level eal ON eal.ID=sgr.Access_level_ID
		JOIN se_access_type eat ON eat.ID=sgr.Access_type_ID
	WHERE su.ID=$1
		AND su.User_status_ID=1
		AND er.Name=$2
		AND eal.Name=$3
	GROUP BY eat.ID
	LIMIT 1`
)

// CheckAccess Checks the user's access to the requested resource and stores the result of the check
// The query must return only the row based on the highest level of access the user is allowed.
// e.g. the order is Group, Owner, World
// Deny would take preferance over Allow - but we don't have this concept yet.
//
// ????? This may need to be rewritten and called from within each handler. ????????????
// for example...
// CheckAccess Checks that the user is authorised to take this action
// Resource = name of the data resource being accesses being accessed
// Action = type of access request action e.g. view, save, edit, list, delete
//
//	func CheckAccess(UserID, Resource string, Action string) bool {
//			// check that the user has permissions to take the requested action
//			// this might also consider information in record being accessed
//	}
func (h *Handler) CheckAccess(UserID int, ResourceName, ActionName string) (int, error) {
	var err error
	var accessType int
	err = h.appConf.Db.QueryRow(sqlCheckAccess, UserID, ResourceName, ActionName).Scan(&accessType)
	if err != nil { // If the number of rows returned is 0 then user is not authorised to access the resource
		log.Println(debugTag+"AccessRepo.CheckAccess()3 ", "Access denied", "err =", err, "accessType =", accessType, "UserID =", UserID, "ResourceName =", ResourceName, "ActionName =", ActionName)
		return 0, err
	}
	return accessType, nil
}
