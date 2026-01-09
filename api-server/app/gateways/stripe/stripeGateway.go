package stripe

import (
	"encoding/json"
	"log"
	"os"

	"github.com/stripe/stripe-go/v72/client"
)

const debugTag = "stripe."

//https://dashboard.stripe.com/test/dashboard
//https://stripe.com/docs/checkout/quickstart
//https://stripe.com/docs/api/checkout/sessions/create#create_checkout_session-line_items-price_data
//https://stripe.com/docs/api/payment_intents

/*
Payment succeeds  4242 4242 4242 4242
Payment requires authentication  4000 0025 0000 3155
Payment is declined  4000 0000 0000 9995
*/

type Charge struct {
	Amount       int64  `json:"amount"`
	ReceiptEmail string `json:"receiptMail"`
	ProductName  string `json:"productName"`
}

type Gateway struct {
	//appConf    *appCore.Config
	Client *client.API
	Domain string
}

// New creates a new Stripe gateway with the API key from a file
func New(keyFile, domain string) *Gateway {
	key := KeyFromFile(keyFile)
	if key == "" {
		log.Printf(debugTag + "New() WARNING: Stripe key is empty, payment features will not work")
	}

	return &Gateway{
		//appConf:    appConf,
		Client: client.New(key, nil),
		Domain: domain,
	}
}

// NewFromKey creates a new Stripe gateway with a direct API key
func NewFromKey(apiKey, domain string) *Gateway {
	if apiKey == "" {
		log.Printf(debugTag + "NewFromKey() WARNING: Stripe key is empty, payment features will not work")
	}

	return &Gateway{
		//appConf:    appConf,
		Client: client.New(apiKey, nil),
		Domain: domain,
	}
}

// KeyFromFile reads the Stripe API key from a JSON file
func KeyFromFile(file string) string {
	f, err := os.Open(file)
	if err != nil {
		log.Printf(debugTag+"KeyFromFile()1 error opening file: %v", err)
		return ""
	}
	defer f.Close()

	var data map[string]string
	err = json.NewDecoder(f).Decode(&data)
	if err != nil {
		log.Printf(debugTag+"KeyFromFile()2 error decoding JSON: %v", err)
		return ""
	}

	if key, ok := data["key"]; ok {
		return key
	}

	log.Printf(debugTag + "KeyFromFile()3 'key' field not found in JSON")
	return ""
}
