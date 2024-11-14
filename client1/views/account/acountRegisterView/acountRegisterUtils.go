package acountRegisterView

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
