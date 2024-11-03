package viewAccountLogin

import (
	"log"
	"math/big"
	"time"

	"github.com/1Password/srp"
)

//const debugTag = "viewLogin."

//Add some sort of timeout on this process ?????????????????????
//Either via context or go routine?????

func (s *LogonForm) authProcess() {
	s.getSalt()
}

// getSalt gets the salt from the server (step 1)
func (s *LogonForm) getSalt() {
	//var err error
	//Get salt from server
	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getSalt()2 success: ", "err =", err, "s.Item =", s.Item) //Log the error in the browser
		}
		s.getServerVerify()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getSalt()3 fail: ", "err =", err, "s.Item =", s.Item) //Log the error in the browser
		//Display message  to user ??????????????
		s.loginErr(err)
	}

	s.Dispatcher.Dispatch(&storeUserAuth.GetSalt{Time: time.Now(), Item: &s.Item, CallbackSuccess: success, CallbackFail: fail})
}

// getServerVerify creates clientEphemeralPublicKey, sends it to the server to retrieve the ServerEphemeralPublicKey and a Proof token (step 2)
func (s *LogonForm) getServerVerify() {
	//var err error
	var A *big.Int

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()2 success: ", "err =", err, "s.Item =", s.Item, "A =", A) //Log the error in the browser
		}
		s.checkServerKey()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"LogonForm.getServerVerify()3 fail: ", "err =", err, "s.Item =", s.Item, "A =", A) //Log the error in the browser
	}

	kdf := srp.KDFRFC5054(s.Item.Salt, s.Item.UserName, s.Item.Password) // Really. Don't use this KDF
	s.srpClient = srp.NewSRPClient(srp.KnownGroups[s.Group], kdf, nil)

	//Fetch client ephemeral public key
	A = s.srpClient.EphemeralPublic()

	//Send the client ephemeral public key to the server
	s.Dispatcher.Dispatch(&storeUserAuth.GetServerVerify{Time: time.Now(), Item: &s.Item, A: A, CallbackSuccess: success, CallbackFail: fail})
	//}
}

// checkServerKey (client) receives B from the server and sets client copy of B (step 3)
func (s *LogonForm) checkServerKey() {
	var err error

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()2 success: ", "err =", err, "s.Item =", s.Item) //Log the error in the browser
		}
		s.loginOk(err)
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()3 fail: ", "err =", err, "s.Item =", s.Item) //Log the error in the browser
	}

	// Once the client receives B from the server it can set client copy of B.
	// Client should check error status here as defense against
	// a malicious B sent from server
	if err = s.srpClient.SetOthersPublic(s.Item.Server.B); err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()4 fail: ", "err =", err, "s.Item =", s.Item) //Log the error in the browser
		// The process has failed
		//log.Fatalf("%v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()4 fail: ", "err =", err, "s.Item =", s.Item)
	}

	// client can now make the session key
	clientKey, err := s.srpClient.Key()
	if err != nil || clientKey == nil {
		log.Printf(debugTag+"LogonForm.checkServerKey()6 Fatal: something went wrong making server key: %s", err)
	}

	// client tests tests that the server sent a good proof
	if !s.srpClient.GoodServerProof(s.Item.Salt, s.Item.UserName, s.Item.Server.Proof) {
		// Client must bail and not send a its own proof back to the server
		log.Fatalf(debugTag+"LogonForm.checkServerKey()7 Fatal: bad proof from server: salt=%s, username=%s, proof=%v", s.Item.Salt, s.Item.UserName, s.Item.Server.Proof)
		return
	}

	// Only after having a valid server proof will the client construct its own proof
	clientProof, err := s.srpClient.ClientProof()
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"LogonForm.checkServerKey()8: ", "err =", err, "clientProof", clientProof, "s.Item =", s.Item) //Log the error in the browser
	}

	// client sends its proof to the server. Server checks
	s.Dispatcher.Dispatch(&storeUserAuth.PostProof{Time: time.Now(), ClientProof: clientProof, Token: s.Item.Server.Token, UserName: s.Item.UserName, CallbackSuccess: success, CallbackFail: fail})
}
