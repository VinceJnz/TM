package handlerWebAuthn

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
)

const debugTag = "handlerWebAuthn."

//var webAuthnInstance *webauthn.WebAuthn

type Handler struct {
	appConf  *appCore.Config
	webAuthn *webauthn.WebAuthn
}

func New(appConf *appCore.Config) *Handler {
	webAuthnInstance, err := webauthn.New(&webauthn.Config{
		RPDisplayName: appConf.Settings.AppTitle,
		RPID:          appConf.Settings.Host,
		RPOrigins:     []string{"https://" + appConf.Settings.Host + ":" + appConf.Settings.PortHttp},
	})
	if err != nil {
		panic("failed to create WebAuthn from config: " + err.Error())
	}
	return &Handler{
		webAuthn: webAuthnInstance,
		appConf:  appConf,
	}
}

// Registration (Begin)
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromRegistrationRequest(r)
	if err != nil {
		http.Error(w, "Failed to get user from request", http.StatusBadRequest)
		return
	}
	options, sessionData, err := h.webAuthn.BeginRegistration(
		user,
	)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		return
	}
	// Store sessionData in your session store
	// TODO: Implement your own session management here
	_ = sessionData
	json.NewEncoder(w).Encode(options)
}

// Registration (Finish)
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromRegistrationRequest(r)
	if err != nil {
		http.Error(w, "Failed to get user from request", http.StatusBadRequest)
		return
	}
	// TODO: Retrieve sessionData from your session store
	var sessionData webauthn.SessionData
	//credential, err := webAuthnInstance.FinishRegistration(user, r, sessionData)
	credential, err := h.webAuthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		http.Error(w, "Failed to finish registration", http.StatusBadRequest)
		return
	}
	user.Credentials = append(user.Credentials, *credential)
	h.saveUser(user)
	w.WriteHeader(http.StatusOK)
}

// Login (Begin)
func (h *Handler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromAuthenticatedRequest(r)
	if err != nil {
		http.Error(w, "Failed to get user from request", http.StatusBadRequest)
		return
	}
	options, sessionData, err := h.webAuthn.BeginLogin(user)
	if err != nil {
		http.Error(w, "Failed to begin login", http.StatusInternalServerError)
		return
	}
	// Store sessionData in your session store
	// TODO: Implement your own session management here
	_ = sessionData
	json.NewEncoder(w).Encode(options)
}

// Login (Finish)
func (h *Handler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromAuthenticatedRequest(r)
	if err != nil {
		http.Error(w, "Failed to get user from request", http.StatusBadRequest)
		return
	}
	// TODO: Retrieve sessionData from your session store
	var sessionData webauthn.SessionData
	//_, err := webAuthnInstance.FinishLogin(user, r, sessionData)
	_, err = h.webAuthn.FinishLogin(user, sessionData, r)
	if err != nil {
		http.Error(w, "Failed to finish login", http.StatusBadRequest)
		return
	}
	h.setUserAuthenticated(w, user)
	w.WriteHeader(http.StatusOK)
}

// getUserFromRegistrationRequest You must implement getUserFromRequest, saveUser, setUserAuthenticated, and session management.
func (h *Handler) getUserFromRegistrationRequest(r *http.Request) (*models.User, error) {
	//Read the data from the web form or JSON body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, err // handle error appropriately
	}

	return &user, nil
	//return &models.User{
	//	ID:       1,
	//	Username: "testuser",
	//	Name:     "Test User",
	//	Email:    zero.NewString("testuser@example.com", true),
	//}, nil
}

// getUserFromRegistrationRequest You must implement getUserFromRequest, saveUser, setUserAuthenticated, and session management.
func (h *Handler) getUserFromAuthenticatedRequest(r *http.Request) (*models.User, error) {
	token, err := h.extractUserTokenFromSession(r)
	if err != nil {
		return nil, err
	}
	user, err := h.UserReadQry(token.UserID)
	if err != nil {
		return nil, err // handle error appropriately
	}
	return &user, nil
}

// extractUserTokenFromSession is a function to use the session id to retrieve user token.
func (h *Handler) extractUserTokenFromSession(r *http.Request) (*models.Token, error) {
	var err error
	var token models.Token
	sessionToken, err := r.Cookie("session")
	if err == http.ErrNoCookie { // If there is no session cookie
		log.Println(debugTag+"Handler.extractUserTokenFromSession()2 - Authentication required ", "sessionToken=", sessionToken, "err =", err)
		return nil, err
	} else { // If there is a session cookie try to find it in the repository
		token, err = h.FindSessionToken(sessionToken.Value) //This succeeds if the cookie is in the DB and the user is current
		if err != nil {                                     // could not find user sessionToken so token is not valid
			log.Println(debugTag+"Handler.extractUserTokenFromSession()3 - Not authorised ", "token =", token, "err =", err)
			return nil, err
		}
	}
	return &token, nil
}

// func (h *Handler) saveUser(user *User) {
func (h *Handler) saveUser(record *models.User) {
	// Save user to your database
	h.UserWriteQry(*record) // Assuming UserWriteQry is implemented to save the user
}

// ?????????????????????????????????? ?????????????????????????????????
func (h *Handler) setUserAuthenticated(w http.ResponseWriter, user *models.User) {
	http.SetCookie(w, &http.Cookie{
		Name:  "session_id",
		Value: "some-session-id", // Replace with actual session ID logic
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User authenticated"})
}

const (
	//Finds only valid cookies where the user account is current
	//if the user account is disabled or set to new it will not return the cookie
	//if the cookie is not valid it will not return the cookie.
	sqlFindSessionToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.token_valid, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		JOIN st_users stu ON stu.ID=stt.User_ID
	WHERE stt.token=$1 AND stt.Name='session' AND stt.token_valid AND stu.user_account_status_ID=$2`

	//Finds valid tokens where user account exists and the token name is the same as the name passed in
	sqlFindToken = `SELECT stt.ID, stt.User_ID, stt.Name, stt.token, stt.token_valid, stt.Valid_From, stt.Valid_To
	FROM st_token stt
		JOIN st_users stu ON stu.ID=stt.User_ID
	WHERE stt.token=$1 AND stt.Name=$2 AND stt.token_valid`
)

// FindSessionToken using the session cookie string find session cookie data in the DB and return the token item
// if the cookie is not found return the DB error
func (h *Handler) FindSessionToken(cookieStr string) (models.Token, error) {
	var err error
	result := models.Token{}
	result.TokenStr.SetValid(cookieStr)
	//err = r.DBConn.QueryRow(sqlFindCookie, result.CookieStr).Scan(&result.ID, &result.UserID, &result.Name, &result.CookieStr, &result.Valid, &result.ValidID, &result.ValidFrom, &result.ValidTo)
	err = h.appConf.Db.QueryRow(sqlFindSessionToken, result.TokenStr, handlerUserAccountStatus.AccountActive).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.Valid, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"FindSessionToken()2", "err =", err, "sqlFindSessionToken =", sqlFindSessionToken, "result =", result)
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
	err = h.appConf.Db.QueryRow(sqlFindToken, result.TokenStr, result.Name).Scan(&result.ID, &result.UserID, &result.Name, &result.TokenStr, &result.Valid, &result.ValidFrom, &result.ValidTo)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"FindToken()2", "err =", err, "sqlFindToken =", sqlFindToken, "result =", result)
		return result, err
	}
	return result, nil
}
