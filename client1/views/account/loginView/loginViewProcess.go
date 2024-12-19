package loginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"math/big"
	"net/http"

	"github.com/1Password/srp"
)

//const debugTag = "viewLogin."

//Add some sort of timeout on this process ?????????????????????
//Either via context or go routine?????

func (editor *ItemEditor) authProcess() {
	// Next process step
	editor.getSalt(editor.CurrentRecord.Username)
}

// getSalt gets the salt from the server (step 1)
func (editor *ItemEditor) getSalt(username string) {
	//Get salt from server
	var salt []byte

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getSalt()3 success: ", "err =", err, "username =", username, "salt =", salt) //Log the error in the browser
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		}
		editor.CurrentRecord.Salt = salt // Save the salt to the current record

		// Next process step
		editor.getServerVerify(username, editor.CurrentRecord.Password, salt)
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getSalt()4 fail: ", "err =", err, "username =", username) //Log the error in the browser
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		//Display message  to user ??????????????
		//editor.onCompletionMsg(debugTag + "getSalt()1 " + err.Error())
	}
	//log.Printf(debugTag+"LogonForm.getSalt()1 CurrentRecord: %+v, url: %v", editor.CurrentRecord, ApiURL+"/"+username+"/salt/")

	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+username+"/salt/", &salt, nil, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
	//editor.Dispatcher.Dispatch(&storeUserAuth.GetSalt{Time: time.Now(), Item: &editor.CurrentRecord, CallbackSuccess: success, CallbackFail: fail})
}

// getServerVerify creates clientEphemeralPublicKey, sends it to the server to retrieve the ServerEphemeralPublicKey and a Proof token (step 2)
func (editor *ItemEditor) getServerVerify(username string, password string, salt []byte) {
	var A *big.Int
	var ServerVerifyRecord ServerVerify

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()1 success: ", "err =", err, "editor.Children =", editor.Children, "A =", A) //Log the error in the browser
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		}
		// Next process step
		editor.checkServerKey(username, ServerVerifyRecord)
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()2 fail: ", "err =", err, "editor.Children =", editor.Children, "A =", A) //Log the error in the browser
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		//editor.onCompletionMsg(debugTag + "getSalt()1 " + err.Error())
	}

	// WARNING ***********************************************************************************************************************************************************************************
	kdf := srp.KDFRFC5054(salt, username, password) // WARNING !!!!!!!!!!!!!!!!!!!! Really. Don't use this KDF !!!!!!!!!!!!!!!!!!!!
	editor.Children.SrpClient = srp.NewSRPClient(srp.KnownGroups[editor.Children.SrpGroup], kdf, nil)
	// WARNING ***********************************************************************************************************************************************************************************

	//Fetch client ephemeral public key
	A = editor.Children.SrpClient.EphemeralPublic()
	byteStrA, _ := A.MarshalText()

	//log.Printf(debugTag+"LogonForm.getServerVerify()3 CurrentRecord=%+v, username=%+v, url=%v, byteStrA=%v, A=%v", editor.CurrentRecord, username, ApiURL+"/"+username+"/key/"+string(byteStrA), byteStrA, A)

	go func() {
		//log.Printf(debugTag+"LogonForm.getServerVerify()4 CurrentRecord=%+v, username=%+v, url=%v", editor.CurrentRecord, username, ApiURL+"/"+username+"/key/"+string(byteStrA))

		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+username+"/key/"+string(byteStrA), &ServerVerifyRecord, nil, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
	//editor.Dispatcher.Dispatch(&storeUserAuth.GetServerVerify{Time: time.Now(), Item: &editor.CurrentRecord, A: A, CallbackSuccess: success, CallbackFail: fail})
}

// checkServerKey (client) receives B from the server and sets client copy of B (step 3)
func (editor *ItemEditor) checkServerKey(username string, serverVerifyRecord ServerVerify) {
	var err error
	var ClientVerifyRecord ClientVerify
	//var username string

	//editor.Children.ServerVerifyRecord = serverVerifyRecord

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()2 success: ", "err=", err, "s.Item=", editor.CurrentRecord) //Log the error in the browser
		}
		//editor.Children.ClientVerifyRecord = ClientVerifyRecord // Save the ClientVerifyRecord - This is probably not required ???????????????
		log.Printf("%v %v %v %v %+v %v %+v %v %v", debugTag+"LogonForm.checkServerKey()0 success: ", "err=", err, "editor.CurrentRecord=", editor.CurrentRecord, "editor.Children=", editor.Children, "username=", username) //Log the error in the browser
		// Next process step
		editor.loginComplete(username)
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()3 fail: ", "err=", err, "editor.Children=", editor.Children) //Log the error in the browser
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		//editor.onCompletionMsg(debugTag + "getSalt()1 " + err.Error())
	}

	// Once the client receives B from the server it can set client copy of B.
	// Client should check error status here as defense against
	// a malicious B sent from server
	if err = editor.Children.SrpClient.SetOthersPublic(serverVerifyRecord.B); err != nil {
		// The process has failed
		//log.Fatalf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()4 fail: ", "err =", err, "s.Item =", s.Item)
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()4 fail: ", "err=", err, "editor.Children=", editor.Children) //Log the error in the browser
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		return
	}

	// client can now make the session key
	clientKey, err := editor.Children.SrpClient.Key()
	if err != nil || clientKey == nil {
		log.Printf(debugTag+"LogonForm.checkServerKey()6 Fatal: something went wrong making client session key: %s", err)
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		return
	}

	// client tests tests that the server sent a good proof
	if !editor.Children.SrpClient.GoodServerProof(editor.CurrentRecord.Salt, editor.CurrentRecord.Username, serverVerifyRecord.Proof) {
		// Client must bail and not send a its own proof back to the server
		//log.Fatalf(debugTag+"LogonForm.checkServerKey()7 Fatal: bad proof from server: CurrentRecord=%+v, serverVerifyRecord=%+v", editor.CurrentRecord, serverVerifyRecord)
		log.Printf(debugTag+"LogonForm.checkServerKey()7 Fatal: bad proof from server: CurrentRecord=%+v, serverVerifyRecord=%+v", editor.CurrentRecord, serverVerifyRecord)
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		return
	}

	// Only after having a valid server proof will the client construct its own proof
	clientProof, err := editor.Children.SrpClient.ClientProof()
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()8: ", "err =", err, "clientProof", clientProof, "s.Item =", editor.CurrentRecord) //Log the error in the browser
	}

	ClientVerifyRecord.UserName = username
	ClientVerifyRecord.Proof = clientProof
	ClientVerifyRecord.Token = serverVerifyRecord.Token
	//log.Printf("%v %v %v %v %+v %v %+v", debugTag+"LogonForm.checkServerKey()9 ", "err =", err, "editor.Children =", editor.Children, "editor.CurrentRecord=", editor.CurrentRecord) //Log the error in the browser

	// client sends its proof to the server. Server checks
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodPost, ApiURL+"/proof/", &username, &ClientVerifyRecord, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
	//editor.Dispatcher.Dispatch(&storeUserAuth.PostProof{Time: time.Now(), ClientProof: clientProof, Token: editor.Children.ServerVerifyRecord.Token, UserName: editor.CurrentRecord.Username, CallbackSuccess: success, CallbackFail: fail})
}

func (editor *ItemEditor) loginComplete(username string) {
	// Need to do something here to signify the login being successful!!!!
	//log.Printf("%v %v %v %v %+v %v %+v", debugTag+"loginComplete()1 ", "username =", username, "editor.Children =", editor.Children, "editor.CurrentRecord =", editor.CurrentRecord) //Log the error in the browser
	editor.onCompletionMsg(debugTag + "Login successfully completed: " + username)
	editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: debugTag, Data: username})
}
