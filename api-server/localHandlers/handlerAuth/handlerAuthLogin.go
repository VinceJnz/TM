package handlerAuth

import (
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/1Password/srp"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

//const debugTag = "handlerAuth."

//*************************************************************************************************
// Login (authenticate process)
//*************************************************************************************************

// AuthGetSalt (step1) sends the user salt to the client
func (h *Handler) AuthGetSalt(w http.ResponseWriter, r *http.Request) {
	var err error
	var username string
	var user models.User

	vars := mux.Vars(r)
	username = vars["username"]

	//Get the salt for the user
	user, err = h.GetUserSalt(username)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthGetSalt()1 ", "err =", err, "user =", user)
		return
	}

	switch models.AccountStatus(user.AccountStatusID.ValueOrZero()) {
	case models.AccountCurrent:
		//salt stored by the server is sent to the client
		json.NewEncoder(w).Encode(user.Salt)
	case models.AccountResetRequired:
		//Send message requiring the user to reset the password
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Password reset required."))
	default:
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthGetSalt()2 ", "err =", err, "user =", user)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Not authorized"))
	}
}

// AuthGetKeyB (step2) creates the ServerEphemeralPublicKey (B) and sends it to the client
// It requires the clientEphemeralPublicKey (A) and an input
func (h *Handler) AuthGetKeyB(w http.ResponseWriter, r *http.Request) {
	var err error
	//var username string
	var user models.User
	var A = &big.Int{}
	var ServerVerify serverVerify
	var group = srp.RFC5054Group3072 //?????????? This needs to be managed at run time ??????????????

	vars := mux.Vars(r)
	strA := vars["A"]
	user.Username = vars["username"]

	//Get store user auth info (salt, etc...)
	user, err = h.GetUserAuth(user.Username)
	if err != nil {
		log.Println(debugTag + "Handler.AuthGetKeyB()5 Fatal: can't retrieve user auth")
		return
	}

	//Create a server instance
	server := srp.NewSRPServer(srp.KnownGroups[group], user.Verifier, nil)
	if server == nil {
		log.Println(debugTag + "Handler.AuthGetKeyB()6 Couldn't set up server")
		return
	}

	//store the server in a pool - it gets used later in the authentication process so it needs to be stored in temp memory
	//The token needs to be sent ot the client so that the server can be recovered later in the auth process
	token := uuid.NewV4().String()
	h.PoolAdd(token, user.ID, server)
	ServerVerify.Token = token

	//ctx := context.WithValue(context.Background(), token, server)
	//ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	go h.AuthCancel(token) //Remove the pool item after a timeout

	// The server will get A (clients ephemeral public key) from the client
	// which the server will set using SetOthersPublic

	// Server MUST check error status here as defense against
	// a malicious A sent by client.
	A.UnmarshalText([]byte(strA))
	if err = server.SetOthersPublic(A); err != nil {
		log.Printf(debugTag+"Handler.AuthGetKeyB()7 Fatal: getting client key: %s\n", err)
		return
	}

	// server sends its ephemeral public key, B, to client
	// client sets it as others public key.
	if ServerVerify.B = server.EphemeralPublic(); ServerVerify.B == nil {
		log.Printf(debugTag + "Handler.AuthGetKeyB()8 server couldn't make B")
		return
	}

	// server can now make the session key.
	serverKey, err := server.Key()
	if err != nil || serverKey == nil {
		log.Printf(debugTag+"Handler.AuthGetKeyB()9 Fatal: something went wrong making server key: %s\n", err)
		return
	}

	// Server computes a proof, and sends it to the client
	ServerVerify.Proof, err = server.M(user.Salt, user.Username)
	if err != nil {
		log.Fatalf(debugTag+"Handler.AuthGetKeyB()10 Fatal: something went wrong making server proof: %s\n", err)
		return
	}

	//server publicKey(B), Proof and a Token is sent to client
	json.NewEncoder(w).Encode(ServerVerify)
}

// AuthCheckClientProof (step3) checks the client proof against the server's stored info
// The client provides ????
// let the client know if it was successful, etc... (i.e. The client is authenticated)
func (h *Handler) AuthCheckClientProof(w http.ResponseWriter, r *http.Request) {
	var err error
	var user models.User
	var ClientVerify clientVerify
	var group = srp.RFC5054Group3072 //?????????? This needs to be managed at run time ??????????????

	//Process the web data
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(debugTag+"Handler.AuthCheckClientProof()2 ", "err =", err, "r.PostForm =", r.PostForm, "body =", string(body))
		status, err := helpers.SqlErr(err)
		http.Error(w, err.Error(), status)
		return
	}

	//Read the data from the web form
	err = json.Unmarshal(body, &ClientVerify)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthCheckClientProof()4 ", "err =", err, "body =", string(body))
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	//Recover server instance from pool and delete it from the pool (it only gets used once in this process)
	authItem := h.PoolGet(ClientVerify.Token)
	h.PoolDelete(ClientVerify.Token)
	userID := authItem.userID
	server := authItem.serverSRP
	if server == nil {
		log.Printf("%v %v %v %v %v", debugTag+"Handler.AuthCheckClientProof()6 Fatal: Couldn't set up server", "group =", group, "ClientVerify =", ClientVerify)
		return
	}

	// Server checks client proof
	if !server.GoodClientProof([]byte(ClientVerify.Proof)) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Please provide authentication for this site"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorised."))

		log.Printf("%v %v %+v", debugTag+"Handler.AuthCheckClientProof()7 Fatal: bad proof from client", "ClientVerify =", ClientVerify)
		return
	}

	//Authentication successful
	//Create and store a new cookie
	sessionToken, err := h.createSessionToken(userID, r.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create cookie"))
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.AuthCheckClientProof()8: Failed to create cookie, createSessionToken fail", "", err, "userID =", userID, "r.RemoteAddr =", r.RemoteAddr)
		return
	}

	//h.srvc.User.ReadDB(userID, &user) //Fetch the user details
	user, err = h.UserReadQry(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to retreive user details"))
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.AuthCheckClientProof()9: Failed to retreive user details", "", err, "userID =", userID, "r.RemoteAddr =", r.RemoteAddr)
		return
	}

	// If all okay we can let the user know
	http.SetCookie(w, sessionToken)
	w.WriteHeader(http.StatusOK)
	//w.Write([]byte("Login successful"))
	json.NewEncoder(w).Encode(user.Username) // returns "username" in the response body
}

// AuthCancel this is used to cancel the Auth process.
// Could use a context.WithCancel here ??????? to be invesitgated later. ?????????????
func (h *Handler) AuthCancel(token string) {
	time.Sleep(15 * time.Second)
	if _, ok := h.Pool[token]; ok {
		h.PoolDelete(token)
		log.Printf(debugTag + "Handler.AuthCancel()1 ****** Auth timed out: Pool server deleted ********")
	}
}

// createSessionToken store it in the DB and in the session struct and return *http.Token
func (h *Handler) createSessionToken(userID int, host string) (*http.Cookie, error) {
	var err error
	//expiration := time.Now().Add(365 * 24 * time.Hour)
	sessionToken := &http.Cookie{
		Name:  "session",
		Value: uuid.NewV4().String(),
		Path:  "/",
		//Domain:     "",
		//Expires:    time.Time{},
		//RawExpires: "",
		//MaxAge:     0,
		Secure:   false,
		HttpOnly: false,
		//SameSite: http.SameSiteNoneMode,
		SameSite: http.SameSiteStrictMode,
		//Raw:        "",
		//Unparsed:   []string{},
	}
	// Store the session cookie for the user in the database
	tokenItem := models.Token{}
	tokenItem.UserID = userID
	tokenItem.Name.SetValid(sessionToken.Name)
	tokenItem.Host.SetValid(host)
	tokenItem.TokenStr.SetValid(sessionToken.Value)
	tokenItem.ValidID.SetValid(1)
	tokenItem.ValidFrom.SetValid(time.Now())
	tokenItem.ValidTo.SetValid(time.Now().Add(24 * time.Hour))

	tokenItem.ID, err = h.TokenWriteQry(tokenItem)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()1 Fatal: createSessionToken fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
	} else {
		err = h.TokenCleanOld(tokenItem.ID)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()2: Token CleanOld fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
		}
		h.TokenCleanExpired()
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v", debugTag+"Handler.createSessionToken()3: Token CleanExpired fail", "err =", err, "UserID =", userID, "tokenItem =", tokenItem)
		}
	}
	return sessionToken, err
}
