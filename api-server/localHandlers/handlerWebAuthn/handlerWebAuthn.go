package handlerWebAuthn

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/webAuthnPool"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
)

const debugTag = "handlerWebAuthn."

type Handler struct {
	appConf  *appCore.Config
	webAuthn *webauthn.WebAuthn // webAuthn instance for handling WebAuthn operations
	Pool     *webAuthnPool.Pool // Uncomment if you want to use a pool for session data
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
	tempSessionToken, err := createTemporaryToken("temp_session_token", h.appConf.Settings.Host)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		return
	}
	h.Pool.Add(tempSessionToken.Value, user, sessionData, 15) // Assuming you have a pool to manage session data
	// add the session token to the response
	http.SetCookie(w, tempSessionToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// Registration (Finish)
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	var sessionData webauthn.SessionData
	var user *models.User                                   //webauthn.User
	tempSessionToken, err := r.Cookie("temp_session_token") // Retrieve the session cookie
	if err != nil {
		switch err {
		case http.ErrNoCookie: // If there is no session cookie
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			log.Println(debugTag+"Handler.FinishRegistration()1 - Authentication required ", "sessionToken=", tempSessionToken, "err =", err)
			return
		default: // If there is some other error
			http.Error(w, "Error retrieving session cookie", http.StatusInternalServerError)
			log.Println(debugTag+"Handler.FinishRegistration()2 - Error retrieving session cookie ", "sessionToken=", tempSessionToken, "err =", err)
			return
		}
	}
	poolItem := h.Pool.Get(tempSessionToken.Value) // Assuming you have a pool to manage session data
	sessionData = *poolItem.SessionData            // Retrieve session data from the pool
	user = poolItem.User                           // Set the user data from the pool

	credential, err := h.webAuthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		http.Error(w, "Failed to finish registration", http.StatusBadRequest)
		return
	}
	//TODO: Save the credential to the user in your database
	// this requires serializing the credential to store it in the database
	// For example, you might use json.Marshal to convert the credential to JSON

	creds := user.WebAuthnCredentials()
	creds = append(creds, *credential) // Append the new credential to the user's credentials

	// Serialize the credentials to JSON or any other format you use in your database
	credsJSON, err := json.Marshal(creds)
	if err != nil {
		http.Error(w, "Failed to serialize credentials", http.StatusInternalServerError)
		return
	}
	user.Credentials = credsJSON // Update the user's credentials

	h.saveUser(user)
	w.WriteHeader(http.StatusOK)
}

// BeginLogin starts the login process by initiating a WebAuthn login request.
// This first step reads the user name provided in the request and to retrieve the user credentials from the database.
// It then generates the options for the WebAuthn login process, creates a temporary session token.
// The temporary session token is stored in a session pool for later retrieval.
// It then sends the WebAuthn login options and a temporary session token as a cookie to the client.
// The client will then use this session token to complete the login process in the FinishLogin step.
func (h *Handler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	name := handlerStandardTemplate.GetName(w, r)
	user, err := h.UserNameReadQry(name) // Assuming UserNameQry is implemented to read the user by credentials
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginLogin()1: User not found", "err =", err, "userID =", user.ID, "r.RemoteAddr =", r.RemoteAddr)
		return
	}
	options, sessionData, err := h.webAuthn.BeginLogin(user)
	if err != nil {
		http.Error(w, "Failed to begin login", http.StatusInternalServerError)
		return
	}
	tempSessionToken, err := createTemporaryToken("temp_session_token", h.appConf.Settings.Host)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		return
	}
	h.Pool.Add(tempSessionToken.Value, &user, sessionData, 15) // Assuming you have a pool to manage session data
	// add the session token to the response
	http.SetCookie(w, tempSessionToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// FinishLogin completes the login process by verifying the user's credentials and creating a session.
// It retrieves the session data from the session pool using the temporary session token.
// If the session token is not found or invalid, it returns an error.
// If the session token is valid, it calls the webauthn.FinishLogin method to verify the user's credentials.
// If the verification is successful, it sets the user as authenticated and creates a session cookie.
// If the verification fails, it returns an error.
// This function is called after the client has provided the user's credentials in response to the BeginLogin request.
func (h *Handler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	var sessionData webauthn.SessionData
	var user *models.User                                   //webauthn.User
	tempSessionToken, err := r.Cookie("temp_session_token") // Retrieve the session cookie
	if err != nil {
		switch err {
		case http.ErrNoCookie: // If there is no session cookie
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			log.Println(debugTag+"Handler.FinishRegistration()1 - Authentication required ", "sessionToken=", tempSessionToken, "err =", err)
			return
		default: // If there is some other error
			http.Error(w, "Error retrieving session cookie", http.StatusInternalServerError)
			log.Println(debugTag+"Handler.FinishRegistration()2 - Error retrieving session cookie ", "sessionToken=", tempSessionToken, "err =", err)
			return
		}
	}
	poolItem := h.Pool.Get(tempSessionToken.Value) // Assuming you have a pool to manage session data
	sessionData = *poolItem.SessionData            // Retrieve session data from the pool
	user = poolItem.User                           // Set the user data from the pool

	_, err = h.webAuthn.FinishLogin(user, sessionData, r)
	if err != nil {
		http.Error(w, "Failed to finish login", http.StatusBadRequest)
		return
	}
	h.setUserAuthenticated(w, r, user)

	w.WriteHeader(http.StatusOK)
}

/*
func (h *Handler) getCreditentialsFromAuthenticateRequest(r *http.Request) ([]webauthn.Credential, error) {
	// Here you would typically look up the user in your database using the credentials provided
	// For example, you might check the username and password against your user database
	// This is a placeholder implementation; you should replace it with your actual user lookup logic
	// For example, you might check the username and password against your user database

	// Read the data from the web form or JSON body //?????????? needs to be fixed
	var Username string
	if err := json.NewDecoder(r.Body).Decode(&Username); err != nil {
		return nil, err // handle error appropriately
	}

	user, err := h.UserNameReadQry(Username) // Assuming UserReadQry is implemented to read the user by ID
	if err != nil {
		return nil, err // handle error appropriately
	}

	return user.WebAuthnCredentials(), nil
}
*/

// getUserFromRegistrationRequest You must implement getUserFromRequest, saveUser, setUserAuthenticated, and session management.
// TODO: Work out the best way to get the user from the registration request.
func (h *Handler) getUserFromRegistrationRequest(r *http.Request) (*models.User, error) {
	//Read the data from the web form or JSON body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, err // handle error appropriately
	}

	return &user, nil
}

/*
// getUserFromAuthenticatedRequest You must implement getUserFromRequest, saveUser, setUserAuthenticated, and session management.
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
*/

/*
// extractUserTokenFromSession is a function to use the session id to retrieve user token.
func (h *Handler) extractUserTokenFromSession(r *http.Request) (*models.Token, error) {
	var err error
	var token models.Token
	sessionToken, err := r.Cookie("session")
	if err != nil { // If there is an error other than no cookie
		switch err {
		case http.ErrNoCookie: // If there is no session cookie
			log.Println(debugTag+"Handler.extractUserTokenFromSession()1 - Authentication required ", "sessionToken=", sessionToken, "err =", err)
			return nil, err
		default: // If there is some other error
			log.Println(debugTag+"Handler.extractUserTokenFromSession()2 - Error retrieving session cookie ", "sessionToken=", sessionToken, "err =", err)
			return nil, err
		}
	}
	// If there is a session cookie try to find it in the repository
	token, err = h.FindSessionToken(sessionToken.Value) //This succeeds if the cookie is in the DB and the user is current
	if err != nil {                                     // could not find user sessionToken so token is not valid
		log.Println(debugTag+"Handler.extractUserTokenFromSession()3 - Not authorised ", "token =", token, "err =", err)
		return nil, err
	}
	return &token, nil // Session token found and user is current
}
*/

// func (h *Handler) saveUser(user *User) {
func (h *Handler) saveUser(record *models.User) {
	// Save user to your database
	h.UserWriteQry(*record) // Assuming UserWriteQry is implemented to save the user
}

// setUserAuthenticated sets the user as authenticated and creates a session cookie
func (h *Handler) setUserAuthenticated(w http.ResponseWriter, r *http.Request, user *models.User) {
	//Authentication successful
	//Create and store a new cookie
	sessionToken, err := h.createSessionToken(user.ID, r.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create cookie"))
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.AuthCheckClientProof()8: Failed to create cookie, createSessionToken fail", "", err, "userID =", user.ID, "r.RemoteAddr =", r.RemoteAddr)
		return
	}

	http.SetCookie(w, sessionToken) // Set the session cookie in the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User authenticated"})
}
