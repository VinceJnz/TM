package stripe

import (
	"encoding/json"
	"log"
	"os"

	"github.com/stripe/stripe-go/v84"
	//"github.com/stripe/stripe-go/v84/customer"
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
	//Client *client.API
	Client *stripe.Client
	Domain string
}

// New creates a new Stripe gateway with the API key from a file
func New(keyFile, domain string, testMode bool) *Gateway {
	apiKey := KeyFromFile(keyFile)
	if apiKey == "" {
		log.Printf(debugTag + "New() WARNING: Stripe key is empty, payment features will not work")
		// Return a gateway with nil client if no key
		return &Gateway{
			Client: nil,
			Domain: domain,
		}
	}
	//if testMode {
	//	apiKey = "sk_test_" + apiKey
	//} else {
	//	apiKey = "sk_live_" + apiKey
	//}

	// DON'T add sk_test_ prefix - the key already has it!
	// Set the global Stripe key
	stripe.Key = apiKey

	return &Gateway{
		//appConf:    appConf,
		Client: stripe.NewClient(apiKey),
		Domain: domain,
	}
}

// NewFromKey creates a new Stripe gateway with a direct API key
func NewFromKey(apiKey, domain string, testMode bool) *Gateway {
	if apiKey == "" {
		log.Printf(debugTag + "NewFromKey() WARNING: Stripe key is empty, payment features will not work")
		// Return a gateway with nil client if no key
		return &Gateway{
			Client: nil,
			Domain: domain,
		}
	}
	//if testMode {
	//	apiKey = "sk_test_" + apiKey
	//} else {
	//	apiKey = "sk_live_" + apiKey
	//}

	// DON'T add sk_test_ prefix - the key already has it!
	// Set the global Stripe key
	stripe.Key = apiKey

	return &Gateway{
		//appConf:    appConf,
		Client: stripe.NewClient(apiKey),
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
