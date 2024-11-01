package handlerAuth

import (
	"api-server/v2/models"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

//const debugTag = "handlerAuth."

//*************************************************************************************************
// Account update
//*************************************************************************************************

// AuthReset requests a password reset
// The information it needs is the user name
// It responds by emailing an update link and sending a messge for the client UI
func (h *Handler) AuthReset(w http.ResponseWriter, r *http.Request) {
	var err error
	var username string
	var token models.Token

	vars := mux.Vars(r)
	username = vars["username"]

	user := models.UserAuth{}
	user, err = h.GetUserAuth(username)
	if err != nil {
		log.Printf("%v %v %+v", debugTag+"Handler.AuthReset()3 user not found ", "username =", username)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("user name not found"))
		return
	}

	if !((models.AccountStatus(user.AccountStatusID.Int64) == models.AccountCurrent) || (models.AccountStatus(user.AccountStatusID.Int64) == models.AccountResetRequired)) {
		log.Printf("%v %v %+v", debugTag+"Handler.AuthReset()4 user not found ", "username =", username)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("user name not found"))
		return
	}

	token, err = h.createToken(user.ID, r.RemoteAddr, "passwordReset", "5m")
	if err != nil {
		log.Printf("%v %v %+v", debugTag+"Handler.AuthReset()5 failed to generate token ", "username =", username)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to generate token"))
		return
	}

	//Send email to the user
	//h.app.EmailSvc.SendMail(application.Email, "vince.jennings@gmail.com", "Account password reset request", user.DisplayName+": Your account for '"+user.UserName+" <"+user.Email.String+">' has been requested. Paste the token below into the web client\n"+token.TokenStr.ValueOrZero())
	//h.app.EmailSvc.SendMail(user.Email.String, "Account password reset request", user.DisplayName+": A password reset for account '"+user.UserName+" <"+user.Email.String+">' has been requested. Paste the token below into the web client\n"+token.TokenStr.ValueOrZero())
	log.Printf("%v %v %+v %v %+v", debugTag+"Handler.AuthReset()6 ", "username =", username, " token =", token)

	json.NewEncoder(w).Encode("account password reset token has been sent to the registered email address")
}

// AuthUpdate is used to change the Auth of the user account //?????? this needs to be updated so that it uses srp ???????
// The users clicks on a coded link that allows them to update their password
func (h *Handler) AuthUpdate(w http.ResponseWriter, r *http.Request) {
	var err error
	var userC models.User //mdlUser.Item //From client
	var userS models.User //From server
	var token models.Token

	vars := mux.Vars(r)
	tokenStr := vars["token"]

	//Process the web data
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(debugTag+"Handler.AuthUpdate()2 ", "err =", err, "r.PostForm =", r.PostForm, "body =", string(body))
		status, err := ctrlMain.SqlErr(err)
		http.Error(w, err.Error(), status)
		return
	}

	//Read the data from the web form and write it to the mdl strtucture
	err = json.Unmarshal(body, &userC)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthUpdate()4 ", "err =", err, "body =", string(body))
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	token, err = h.srvc.Session.FindToken("passwordReset", tokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthUpdate()5 ", "err =", err, "tokenStr =", tokenStr)
		return
	}

	//Get the server user info
	err = h.srvc.User.ReadDB(token.UserID, &userS)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthUpdate()6 ", "err =", err, "token =", token)
		return
	}

	if !((models.AccountStatus(userS.UserStatusID.Int64) == models.AccountCurrent) || (models.AccountStatus(userS.UserStatusID.Int64) == models.AccountResetRequired)) {
		log.Printf("%v %v %+v", debugTag+"Handler.AuthReset()4 user not found ", "userS.UserName =", userS.Username)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("user name not found"))
		return
	}

	err = h.srvc.Token.DeleteDB(token.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthUpdate()7 ", "err =", err, "token =", token)
	}

	//As an extra: Check the user names match. We don't use this value from the client.
	if userS.Username != userC.Username {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"Handler.AuthUpdate()8 ", "err =", err, "userC =", userC, "userS =", userS)
		return
	}

	//Set the server user account auth info from the info the client provided
	userS.Verifier = userC.Verifier
	userS.Salt = userC.Salt

	err = h.srvc.UserUtils.PutUserAuth(userS)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthUpdate()9 ", "err =", err, "userS =", userS)
		return
	}

	if models.AccountStatus(userS.UserStatusID.ValueOrZero()) == models.AccountResetRequired {
		err = h.srvc.UserStatus.SetStatusID(userS.ID, models.AccountCurrent)
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"Handler.AuthUpdate()10 ", "err =", err, "userS =", userS)
		}
	}

	//h.app.EmailSvc.SendMail(userS.Email.String, "Account password reset notification", userS.DisplayName+": The password for account '"+userS.UserName+" <"+userS.Email.String+">' has been reset.\n")

	json.NewEncoder(w).Encode("Password has been reset")
}
