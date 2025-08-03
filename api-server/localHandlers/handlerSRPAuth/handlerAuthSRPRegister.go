package handlerSRPAuth

import (
	"api-server/v2/dbTemplates/dbAuthTemplate"
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/models"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

//const debugTag = "handlerAuth."

type serverVerify struct {
	B     *big.Int
	Proof []byte
	Token string
}

type clientVerify struct {
	UserName string
	Proof    []byte
	Token    string
}

//*************************************************************************************************
// Register (Account create)
//*************************************************************************************************

// Create creates a new user account and responds with a token that can be used to validate the email address
func (h *Handler) AccountCreate(w http.ResponseWriter, r *http.Request) {
	var err error
	var user models.User

	//Process the web data
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(debugTag+"Handler.AccountCreate()2 ", "err =", err, "r.PostForm =", r.PostForm, "body =", string(body))
		status, err := helpers.SqlErr(err)
		http.Error(w, err.Error(), status)
		return
	}

	//Read the data from the web form and write it to the DB
	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountCreate()4 ", "err =", err, "body =", string(body))
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	//Create the new user record in the DB - this doesn't store the password
	log.Printf("%v %v %+v", debugTag+"Handler.AccountCreate()5", "&user =", &user)
	//user.ID, err = h.UserWriteQry(user)
	user.ID, err = dbAuthTemplate.UserWriteQry(debugTag+"Handler.AccountCreate()5a ", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountCreate()6 ", "err =", err, "user =", user)
		status, err := helpers.SqlErr(err)
		http.Error(w, err.Error(), status)
		return
	}

	//Set the password in the DB
	//err = h.UserAuthUpdate(user)
	err = dbAuthTemplate.UserAuthUpdate(debugTag+"Handler.AccountCreate()6a ", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountCreate()7 ", "err =", err, "user =", user)
		status, err := helpers.SqlErr(err)
		http.Error(w, err.Error(), status)
		return
	}

	//Create and store a token
	token, err := h.createToken(user.ID, r.RemoteAddr, "accountValidation", "24h")
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"Handler.AccountCreate()8 ", "err =", err, "token =", token, "user =", user)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//Send email to the user with the token so that they can validate the new logon ?????????
	//h.app.EmailSvc.SendMail(application.Email, "vince.jennings@gmail.com", "New account validation", "New user: "+user.UserName+" <"+user.Email.String+">\n Validation link "+r.Header["Origin"][0]+h.app.Settings.APIprefix+"/accountvalidate/"+token.TokenStr.String)
	//h.app.EmailSvc.SendMail(user.Email.String, "New account validation", "New user: "+user.UserName+" <"+user.Email.String+">\n Validation link "+r.Header["Origin"][0]+h.app.Settings.APIprefix+"/auth/validate/"+token.TokenStr.String)
	json.NewEncoder(w).Encode("Account created. Check your email for a validation link" + "New user: " + user.Username + " <" + user.Email.String + ">\n Validation link " + r.Header["Origin"][0] + h.appConf.Settings.APIprefix + "/auth/validate/" + token.TokenStr.String)
	//json.NewEncoder(w).Encode("Account created. Check your email for a validation link")
}

// createToken ??????
func (h *Handler) createToken(userID int, host, tokenName string, duration string) (models.Token, error) {
	var err error
	var token models.Token

	d, err := time.ParseDuration(duration)
	if err != nil {
		return token, err
	}
	validFrom := time.Now()
	validTo := validFrom.Add(d)

	value := uuid.NewV4().String()
	token.UserID = userID
	token.Name.SetValid(tokenName)
	token.Host.SetValid(host)
	token.TokenStr.SetValid(value)
	token.Valid.SetValid(true)
	token.ValidFrom.SetValid(validFrom)
	token.ValidTo.SetValid(validTo)

	//token.ID, err = h.srvc.Token.WriteDB(token.ID, &token)
	//token.ID, err = h.TokenWriteQry(token)
	token.ID, err = dbAuthTemplate.TokenWriteQry(debugTag+"Handler.createToken()1 ", h.appConf.Db, token)

	return token, err
}

// AccountValidate the user validates the account by clicking on the link sent to them via email
// This changes the account status and deletes the validaion token so that it can't be used again.
func (h *Handler) AccountValidate(w http.ResponseWriter, r *http.Request) {
	var err error
	var tokenStr string
	//var user models.User

	vars := mux.Vars(r)
	tokenStr = vars["token"]

	//token, err := h.FindToken("accountValidation", tokenStr)
	token, err := dbAuthTemplate.FindToken(debugTag, h.appConf.Db, "accountValidation", tokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountValidate()5 ", "err =", err, "token =", token)
		status, err := helpers.SqlErr(err)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, err.Error(), status)
		return
	}
	//delete/invalidate the token - it is allowed to be used only once.
	//err = h.TokenDeleteQry(token.ID)
	err = dbAuthTemplate.TokenDeleteQry(debugTag+"Handler.AccountValidate()5a ", h.appConf.Db, token.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountValidate()6 ", "err =", err, "token =", token)
		status, err := helpers.SqlErr(err)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, err.Error(), status)
		return
	}

	//Set the user account to verified
	//err = h.srvc.SetUserStatus(token.UserID, mdlUser.Verified)
	//err = h.UserSetStatusID(token.UserID, handlerUserAccountStatus.AccountVerified)
	err = dbAuthTemplate.UserSetStatusID(debugTag+"Handler.AccountValidate()6a ", h.appConf.Db, token.UserID, handlerUserAccountStatus.AccountVerified)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountValidate()7 ", "err =", err, "token =", token)
		status, err := helpers.SqlErr(err)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, err.Error(), status)
		return
	}

	//Get the user details and Send an email to the user
	//user, err := h.UserReadQry(token.UserID)
	user, err := dbAuthTemplate.UserReadQry(debugTag+"Handler.AccountValidate()7a ", h.appConf.Db, token.UserID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountValidate()7a ", "err =", err, "user =", user) // User this for testing
		status, err := helpers.SqlErr(err)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, err.Error(), status)
		return
	}
	if h.appConf.TestMode {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AccountValidate()7a ", "err =", err, "user =", user) // User this for testing
	} else {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"Handler.AccountValidate()7b ", "err =", err, "email to user =", user.Email.String, "New account validated", user.Name+": Your account for '"+user.Username+" <"+user.Email.String+">' has been validated.\nAn administrator will review your account and activate it shortly.") // User this for testing
		//h.app.EmailSvc.SendMail(user.Email.String, "New account validated", user.Name+": Your account for '"+user.Username+" <"+user.Email.String+">' has been validated.\nAn administrator will review your account and activate it shortly.")
	}

	//Notify administrators of the validated accounts
	//adminList, err := h.GetAdminList(1)
	adminList, err := dbAuthTemplate.GetAdminList(debugTag+"Handler.AccountValidate()8 ", h.appConf.Db, 1) // Get the admin list for group 1
	if err == nil {
		for _, adminUser := range adminList {
			log.Printf("%v %v %+v", debugTag+"Handler.AccountValidate()9 ", "adminUser =", adminUser)
			//h.app.EmailSvc.SendMail(adminUser.Email.String, "New account to be activated ", "Hi "+adminUser.DisplayName+",\nPlease check this new user.\n"+user.DisplayName+": '"+user.UserName+" <"+user.Email.String+">'\nPlease review the account, add it to a group and activate it if appropriate.")
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("account validated")
}
