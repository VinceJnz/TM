package appCore

import (
	"api-server/v2/app/gateways/gmail"
	"api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	"api-server/v2/app/gateways/stripe"
	"log"

	"github.com/jmoiron/sqlx"
)

const debugTag = "appCore."

type ContextKey string

func GenerateSessionIDContextKey() ContextKey { // User for generating the context key for passing values via the context (ctx)
	return ContextKey("sessionID")
}

type Config struct {
	Db           *sqlx.DB
	EmailSvc     *gmail.Gateway
	PaymentSvc   *stripe.Gateway
	OAuthSvc     *oAuthGateway.Gateway
	SessionIDKey ContextKey // User for passing the user id value via the context (ctx)
	Settings     settings
	TestMode     bool
}

func New(testMode bool) *Config {
	sessionIDKey := GenerateSessionIDContextKey()
	return &Config{
		Db:           &sqlx.DB{},
		EmailSvc:     &gmail.Gateway{},
		PaymentSvc:   &stripe.Gateway{},
		OAuthSvc:     &oAuthGateway.Gateway{},
		SessionIDKey: sessionIDKey,
		TestMode:     testMode,
	}
}

func (c *Config) Run() {
	var err error
	// Load environment variables
	c.Settings.LoadEnv()

	// Initialize the database connection
	c.Db, err = InitDB()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)

	}
	// Start the email service
	c.EmailSvc = gmail.New(c.Settings.EmailSecret, c.Settings.EmailToken, c.Settings.EmailAddr)

	// Start the payment service
	domain := "https://" + c.Settings.Host + ":" + c.Settings.PortHttps + c.Settings.APIprefix
	c.PaymentSvc = stripe.NewFromKey(c.Settings.PaymentKey, domain, c.TestMode)
	log.Printf(debugTag+"Run() Stripe gateway initialized with domain: %s", domain)
	//go c.PaymentSvc.Start() // No longer needed as we are not using webhooks ??????

	c.OAuthSvc, err = oAuthGateway.NewFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize oAuth Gateway: %v", err)
	}

}

func (c *Config) Close() {
	c.Db.Close()
}

func (c *Config) Access() {

}
