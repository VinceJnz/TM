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
		Db:       &sqlx.DB{},
		EmailSvc: &gmail.Gateway{},
		//PaymentSvc:   &stripe.Gateway{},
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
	//paymentSvc := stripe.New(a.Db, a.WsPool, a.Settings.PaymentKey, "https://"+a.Settings.Host+":"+a.Settings.PortHttps+a.Settings.APIprefix)
	//paymentSvc := stripe.New(a.Db, a.Settings.PaymentKey, "https://"+a.Settings.Host+":"+a.Settings.PortHttps+a.Settings.APIprefix)
	//go c.PaymentSvc.Start()

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
