package srpLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"syscall/js"
)

//const debugTag = "srpLoginView."

//Add some sort of timeout on this process ?????????????????????
//Either via context or go routine?????

/*
func (editor *ItemEditor) authProcess() {
	// Next process step
	editor.getSalt(editor.CurrentRecord.Username)
}

// getSalt gets the salt from the server (step 1)
func (editor *ItemEditor) getSalt(username string) {
	//Get salt from server
	var salt []byte

	success := func(err error, data *httpProcessor.ReturnData) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getSalt()3 success: ", "err =", err, "username =", username, "salt =", salt) //Log the error in the browser
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		}
		editor.CurrentRecord.Salt = salt // Save the salt to the current record

		// Next process step
		editor.getServerVerify(username, editor.CurrentRecord.Password, salt)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getSalt()4 fail: ", "err =", err, "username =", username) //Log the error in the browser
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
	}

	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+username+"/salt/", &salt, nil, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
}

// getServerVerify creates clientEphemeralPublicKey, sends it to the server to retrieve the ServerEphemeralPublicKey and a Proof token (step 2)
func (editor *ItemEditor) getServerVerify(username string, password string, salt []byte) {
	var A *big.Int
	var ServerVerifyRecord ServerVerify

	success := func(err error, data *httpProcessor.ReturnData) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()1 success: ", "err =", err, "editor.Children =", editor.Children, "A =", A) //Log the error in the browser
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		}
		// Next process step
		editor.checkServerKey(username, ServerVerifyRecord)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()2 fail: ", "err =", err, "editor.Children =", editor.Children, "A =", A) //Log the error in the browser
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
	}

	// WARNING ***********************************************************************************************************************************************************************************
	kdf := srp.KDFRFC5054(salt, username, password) // WARNING !!!!!!!!!!!!!!!!!!!! Really. Don't use this KDF !!!!!!!!!!!!!!!!!!!!
	editor.Children.SrpClient = srp.NewSRPClient(srp.KnownGroups[editor.Children.SrpGroup], kdf, nil)
	// WARNING ***********************************************************************************************************************************************************************************

	//Fetch client ephemeral public key
	A = editor.Children.SrpClient.EphemeralPublic()
	byteStrA, _ := A.MarshalText()

	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+username+"/key/"+string(byteStrA), &ServerVerifyRecord, nil, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
}

// checkServerKey (client) receives B from the server and sets client copy of B (step 3)
func (editor *ItemEditor) checkServerKey(username string, serverVerifyRecord ServerVerify) {
	var err error
	var ClientVerifyRecord ClientVerify

	success := func(err error, data *httpProcessor.ReturnData) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()2 success: ", "err=", err, "s.Item=", editor.CurrentRecord) //Log the error in the browser
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		}
		// Next process step
		editor.loginComplete(username)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()3 fail: ", "err=", err, "editor.Children=", editor.Children) //Log the error in the browser
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
	}

	// Once the client receives B from the server it can set client copy of B.
	// Client should check error status here as defense against
	// a malicious B sent from server
	if err = editor.Children.SrpClient.SetOthersPublic(serverVerifyRecord.B); err != nil {
		// The process has failed
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

	// client sends its proof to the server. Server checks
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodPost, ApiURL+"/proof/", &username, &ClientVerifyRecord, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
}

func (editor *ItemEditor) loginComplete(username string) {
	// Need to do something here to signify the login being successful!!!!
	editor.onCompletionMsg(debugTag + "Login successfully completed: " + username)
	editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: debugTag, Data: username})
}
*/

// encodeBase64URL converts raw bytes to base64url (no padding) string
func encodeBase64URL(b []byte) string {
	s := base64.StdEncoding.EncodeToString(b)
	s = strings.TrimRight(s, "=")
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}

// authProcess kicks off the SRP login flow
func (editor *ItemEditor) authProcess() {
	editor.getSalt(editor.CurrentRecord.Username)
}

// getSalt gets the salt from the server (step 1)
func (editor *ItemEditor) getSalt(username string) {
	var salt []byte

	success := func(err error, data *httpProcessor.ReturnData) {
		if err != nil {
			log.Printf("%v LogonForm.getSalt() error: %v, username=%s", debugTag, err, username)
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
			return
		}
		editor.CurrentRecord.Salt = salt
		editor.getServerVerify(username, editor.CurrentRecord.Password, salt)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v LogonForm.getSalt() fail: %v, username=%s", debugTag, err, username)
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
	}

	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+username+"/salt/", &salt, nil, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
}

// getServerVerify creates client ephemeral A using JS, sends it to the server to retrieve B and server proof (step 2)
func (editor *ItemEditor) getServerVerify(username string, password string, salt []byte) {
	var ServerVerifyRecord ServerVerify

	success := func(err error, data *httpProcessor.ReturnData) {
		if err != nil {
			log.Printf("%v getServerVerify success callback error: %v", debugTag, err)
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
			return
		}

		// ServerVerifyRecord received; process B and server proof with JS to compute client proof and verify server proof
		saltB64 := encodeBase64URL(salt)

		// Call JS to compute client proof and verify server proof
		jsFn := js.Global().Get("srpComputeClientProof")
		if jsFn.IsUndefined() || jsFn.IsNull() {
			log.Printf("%v srpComputeClientProof JS function not found", debugTag)
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed (client SRP not available)"})
			return
		}

		// Invoke JS: (username, password, saltB64, B_b64, serverProof_b64)
		res := jsFn.Invoke(username, password, saltB64, ServerVerifyRecord.B, ServerVerifyRecord.Proof)
		if res.IsUndefined() || res.IsNull() {
			log.Printf("%v srpComputeClientProof returned null", debugTag)
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed (SRP computation error)"})
			return
		}

		clientProof := res.Get("proof").String()
		validServer := res.Get("validServer").Bool()
		if !validServer {
			log.Printf("%v srp: server proof invalid", debugTag)
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed (bad server proof)"})
			return
		}

		// Build ClientVerify and send to server
		var ClientVerifyRecord ClientVerify
		ClientVerifyRecord.UserName = username
		ClientVerifyRecord.Proof = clientProof
		ClientVerifyRecord.Token = ServerVerifyRecord.Token

		// send client proof to server
		doneSuccess := func(err error, data *httpProcessor.ReturnData) {
			if err != nil {
				log.Printf("%v send proof success callback error: %v", debugTag, err)
				editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
				return
			}
			editor.loginComplete(username)
		}
		doneFail := func(err error, data *httpProcessor.ReturnData) {
			log.Printf("%v send proof fail: %v", debugTag, err)
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
		}

		go func() {
			editor.updateStateDisplay(viewHelpers.ItemStateFetching)
			editor.client.NewRequest(http.MethodPost, ApiURL+"/proof/", &username, &ClientVerifyRecord, doneSuccess, doneFail)
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
		}()
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v getServerVerify fail: %v", debugTag, err)
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
	}

	// Compute client ephemeral A using JS srpComputeA(username, password, saltB64)
	saltB64 := encodeBase64URL(salt)
	jsA := js.Global().Get("srpComputeA")
	if jsA.IsUndefined() || jsA.IsNull() {
		log.Printf("%v srpComputeA JS function not found", debugTag)
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed (client SRP not available)"})
		return
	}
	A_b64 := jsA.Invoke(username, password, saltB64).String()
	if A_b64 == "" {
		log.Printf("%v srpComputeA returned empty A", debugTag)
		editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed (SRP computation error)"})
		return
	}

	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		// Send A to server; server returns B, proof, token
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+username+"/key/"+A_b64, &ServerVerifyRecord, nil, success, fail)
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}()
}

// loginComplete triggered when login succeeds
func (editor *ItemEditor) loginComplete(username string) {
	editor.onCompletionMsg(debugTag + "Login successfully completed: " + username)
	editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: debugTag, Data: username})
}
