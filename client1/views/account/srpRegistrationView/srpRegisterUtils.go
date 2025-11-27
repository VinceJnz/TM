package srpRegistrationView

/*
import (
	"crypto/rand"
	"log"

	"github.com/1Password/srp"
)

// authCreate choose the RFCgroup, generated a random key, generate the verifier key.
func authCreate(Item TableData) (TableData, error) {
	var err error

	//group has to be negoiated between client and server
	Item.group = srp.RFC5054Group3072

	// Generate 8 bytes of random salt. Be sure to use crypto/rand for all
	// of your random number needs
	Item.Salt = make([]byte, 8)
	if n, err := rand.Read(Item.Salt); err != nil {
		log.Fatal(err)
	} else if n != 8 {
		log.Fatal("failed to generate 8 byte salt")
	}

	// You would use a better Key Derivation Function than this one ??????????????????????????????
	x := srp.KDFRFC5054(Item.Salt, Item.Username, Item.Password) // Really. Don't use this KDF ?????????????????????????????????

	// this is still our first use scenario, but the client needs to create
	// an SRP client to generate the verifier.
	firstClient := srp.NewSRPClient(srp.KnownGroups[Item.group], x, nil)
	if firstClient == nil {
		log.Fatal("couldn't setup client")
	}

	Item.Verifier, err = firstClient.Verifier()
	if err != nil {
		log.Fatal(err)
	}
	return Item, nil
}
*/

import (
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"syscall/js"
)

// authCreate delegates SRP salt+verifier generation to a JS function exposed on the page.
// The page must provide a synchronous JS function computeSRPVerifier(username, password)
// that returns an object { salt: "<base64url>", verifier: "<base64url>" }.
//
// Example JS (must be loaded into the page):
// <script>
//
//	// using an SRP JS library (not provided here) produce salt & verifier as base64url
//	function computeSRPVerifier(username, password) {
//	  const { salt, verifier } = SRPLib.computeVerifier(username, password) // pseudo
//	  return { salt: toBase64Url(saltBytes), verifier: toBase64Url(verifierBytes) }
//	}
//
// </script>
//
// This keeps heavy crypto in the browser JS while returning values to Go.
func authCreate(Item TableData) (TableData, error) {
	// Look up the JS function
	jsFn := js.Global().Get("computeSRPVerifier")
	if jsFn.IsUndefined() || jsFn.IsNull() {
		return Item, errors.New("computeSRPVerifier JS function not found")
	}

	// Call the JS function synchronously: expect { salt: "...", verifier: "..." }
	res := jsFn.Invoke(Item.Username, Item.Password)
	if res.IsUndefined() || res.IsNull() {
		return Item, errors.New("computeSRPVerifier returned null/undefined")
	}

	saltB64 := res.Get("salt").String()
	verifierB64 := res.Get("verifier").String()
	if saltB64 == "" || verifierB64 == "" {
		return Item, errors.New("computeSRPVerifier returned empty salt or verifier")
	}

	// decode base64url function
	dec := func(b64url string) ([]byte, error) {
		// base64url -> std base64
		s := b64url
		// Add padding
		if m := len(s) % 4; m != 0 {
			s += strings.Repeat("=", 4-m)
		}
		s = strings.ReplaceAll(s, "-", "+")
		s = strings.ReplaceAll(s, "_", "/")
		return base64.StdEncoding.DecodeString(s)
	}

	saltBytes, err := dec(saltB64)
	if err != nil {
		log.Printf("authCreate: failed to decode salt: %v", err)
		return Item, err
	}

	verifierBytes, err := dec(verifierB64)
	if err != nil {
		log.Printf("authCreate: failed to decode verifier: %v", err)
		return Item, err
	}

	// Populate Item.Salt and Item.Verifier (big.Int) so existing code continues to work
	Item.Salt = saltBytes
	Item.Verifier = verifierBytes

	// Clear plaintext password from memory if you prefer (optional)
	// Item.Password = ""

	return Item, nil
}
