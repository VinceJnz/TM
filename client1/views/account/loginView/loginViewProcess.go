package loginView

import (
	"client1/v2/app/httpProcessor"
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
	editor.getSalt()
}

// getSalt gets the salt from the server (step 1)
func (editor *ItemEditor) getSalt() {
	//var err error
	//Get salt from server
	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getSalt()3 success: ", "err =", err, "s.Item =", editor.CurrentRecord) //Log the error in the browser
		}
		editor.CurrentRecord.Salt = editor.Children.SrpRecord.Salt
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getSalt()0 success: ", "err =", err, "editor.Children =", editor.Children) //Log the error in the browser
		// Next process step
		editor.getServerVerify()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getSalt()4 fail: ", "err =", err, "s.Item =", editor.CurrentRecord) //Log the error in the browser
		//Display message  to user ??????????????
		editor.onCompletionMsg(debugTag + "getSalt()1 " + err.Error())
	}
	log.Printf(debugTag+"LogonForm.getSalt()1 CurrentRecord: %+v, url: %v", editor.CurrentRecord, editor.baseURL+apiURL+"/"+editor.CurrentRecord.Username+"/salt/")
	username := editor.CurrentRecord.Username
	//if editor.RecordState == RecordStateReloadRequired {
	//	editor.RecordState = RecordStateCurrent
	go func() {
		log.Printf(debugTag+"LogonForm.getSalt()2 CurrentRecord: %+v, username: %+v, url: %v", editor.CurrentRecord, username, editor.baseURL+apiURL+"/"+username+"/salt/")
		//var salt []byte
		editor.updateStateDisplay(ItemStateFetching)
		//httpProcessor.NewRequest(http.MethodGet, editor.baseURL+apiURL+"/"+username+"/salt/", &salt, nil, success, fail)
		//editor.Children.SrpRecord.Salt = salt
		httpProcessor.NewRequest(http.MethodGet, editor.baseURL+apiURL+"/"+username+"/salt/", &editor.Children.SrpRecord.Salt, nil, success, fail)
		editor.updateStateDisplay(ItemStateNone)
	}()
	//}
	log.Printf(debugTag+"LogonForm.getSalt()5 CurrentRecord: %+v, url: %v", editor.CurrentRecord, editor.baseURL+apiURL+"/"+editor.CurrentRecord.Username+"/salt/")

	//editor.Dispatcher.Dispatch(&storeUserAuth.GetSalt{Time: time.Now(), Item: &editor.CurrentRecord, CallbackSuccess: success, CallbackFail: fail})
}

// getServerVerify creates clientEphemeralPublicKey, sends it to the server to retrieve the ServerEphemeralPublicKey and a Proof token (step 2)
func (editor *ItemEditor) getServerVerify() {
	//var err error
	var A *big.Int
	var ServerVerifyRecord ServerVerify

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()1 success: ", "err =", err, "editor.CurrentRecord =", editor.CurrentRecord, "A =", A) //Log the error in the browser
		}
		editor.Children.ServerVerifyRecord = ServerVerifyRecord
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()0 success: ", "err =", err, "editor.Children =", editor.Children, "A =", A) //Log the error in the browser
		// Next process step
		editor.checkServerKey()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()2 fail: ", "err =", err, "editor.CurrentRecord =", editor.CurrentRecord, "A =", A) //Log the error in the browser
		//editor.onCompletionMsg(debugTag + "getSalt()1 " + err.Error())
	}

	// WARNING ***********************************************************************************************************************************************************************************
	kdf := srp.KDFRFC5054(editor.CurrentRecord.Salt, editor.CurrentRecord.Username, editor.CurrentRecord.Password) // WARNING !!!!!!!!!!!!!!!!!!!! Really. Don't use this KDF !!!!!!!!!!!!!!!!!!!!
	editor.SrpClient = srp.NewSRPClient(srp.KnownGroups[editor.Children.Group], kdf, nil)
	// WARNING ***********************************************************************************************************************************************************************************

	//Fetch client ephemeral public key
	A = editor.SrpClient.EphemeralPublic()
	byteStrA, _ := A.MarshalText()

	username := editor.CurrentRecord.Username
	log.Printf(debugTag+"LogonForm.getServerVerify()3 CurrentRecord=%+v, username=%+v, url=%v, byteStrA=%v, A=%v", editor.CurrentRecord, username, editor.baseURL+apiURL+"/"+username+"/key/"+string(byteStrA), byteStrA, A)

	//if editor.RecordState == RecordStateReloadRequired {
	//	editor.RecordState = RecordStateCurrent
	go func() {
		log.Printf(debugTag+"LogonForm.getServerVerify()4 CurrentRecord=%+v, username=%+v, url=%v", editor.CurrentRecord, username, editor.baseURL+apiURL+"/"+username+"/key/"+string(byteStrA))

		editor.updateStateDisplay(ItemStateFetching)
		httpProcessor.NewRequest(http.MethodGet, editor.baseURL+apiURL+"/"+username+"/key/"+string(byteStrA), &ServerVerifyRecord, nil, success, fail)
		editor.updateStateDisplay(ItemStateNone)
	}()
	//}

	//Send the client ephemeral public key to the server
	//editor.Dispatcher.Dispatch(&storeUserAuth.GetServerVerify{Time: time.Now(), Item: &editor.CurrentRecord, A: A, CallbackSuccess: success, CallbackFail: fail})
	//}
}

// checkServerKey (client) receives B from the server and sets client copy of B (step 3)
func (editor *ItemEditor) checkServerKey() {
	var err error
	var ClientVerifyRecord ClientVerify

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()2 success: ", "err=", err, "s.Item=", editor.CurrentRecord) //Log the error in the browser
		}
		editor.Children.ClientVerifyRecord = ClientVerifyRecord
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"LogonForm.checkServerKey()0 success: ", "err=", err, "editor.CurrentRecord=", editor.CurrentRecord, "editor.Children=", editor.Children) //Log the error in the browser
		// Next process step
		// Need to do something here to signify the login being successful!!!!
		editor.onCompletionMsg(debugTag + "checkServerKey()1 " + err.Error())
		//editor.loginOk(err)
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()3 fail: ", "err=", err, "editor.Children=", editor.Children) //Log the error in the browser
		//editor.onCompletionMsg(debugTag + "getSalt()1 " + err.Error())
	}

	// Once the client receives B from the server it can set client copy of B.
	// Client should check error status here as defense against
	// a malicious B sent from server
	if err = editor.SrpClient.SetOthersPublic(editor.Children.ServerVerifyRecord.B); err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()4 fail: ", "err=", err, "editor.Children=", editor.Children) //Log the error in the browser
		// The process has failed
		//log.Fatalf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()4 fail: ", "err =", err, "s.Item =", s.Item)
	}

	// client can now make the session key
	clientKey, err := editor.SrpClient.Key()
	if err != nil || clientKey == nil {
		log.Printf(debugTag+"LogonForm.checkServerKey()6 Fatal: something went wrong making client session key: %s", err)
	}

	// client tests tests that the server sent a good proof
	if !editor.SrpClient.GoodServerProof(editor.CurrentRecord.Salt, editor.CurrentRecord.Username, editor.Children.ServerVerifyRecord.Proof) {
		// Client must bail and not send a its own proof back to the server
		log.Fatalf(debugTag+"LogonForm.checkServerKey()7 Fatal: bad proof from server: CurrentRecord=%+v, ServerVerifyRecord=%+v", editor.CurrentRecord, editor.Children.ServerVerifyRecord)
		return
	}

	// Only after having a valid server proof will the client construct its own proof
	clientProof, err := editor.SrpClient.ClientProof()
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()8: ", "err =", err, "clientProof", clientProof, "s.Item =", editor.CurrentRecord) //Log the error in the browser
	}

	ClientVerifyRecord.UserName = editor.CurrentRecord.Username
	ClientVerifyRecord.Proof = clientProof
	ClientVerifyRecord.Token = editor.Children.ServerVerifyRecord.Token
	editor.Children.ClientVerifyRecord = ClientVerifyRecord
	log.Printf("%v %v %v %v %+v %v %+v", debugTag+"LogonForm.checkServerKey()9 ", "err =", err, "editor.Children =", editor.Children, "editor.CurrentRecord=", editor.CurrentRecord) //Log the error in the browser

	//if editor.RecordState == RecordStateReloadRequired {
	//	editor.RecordState = RecordStateCurrent
	go func() {
		editor.updateStateDisplay(ItemStateFetching)
		httpProcessor.NewRequest(http.MethodPost, editor.baseURL+apiURL+"/proof/", &ClientVerifyRecord, nil, success, fail)
		editor.updateStateDisplay(ItemStateNone)
	}()
	//}

	//if err := s.Client.SendPostRequest("auth/proof/", clientVerify, username, success, fail); err != nil { //Send the REST request

	// client sends its proof to the server. Server checks
	//editor.Dispatcher.Dispatch(&storeUserAuth.PostProof{Time: time.Now(), ClientProof: clientProof, Token: editor.Children.ServerVerifyRecord.Token, UserName: editor.CurrentRecord.Username, CallbackSuccess: success, CallbackFail: fail})
}
