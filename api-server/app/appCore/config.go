package appCore

import (
	"api-server/v2/app/gateways/gmail"
	"api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	"api-server/v2/app/gateways/stripe"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"log"
	"time"

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
	if err = c.Settings.LoadEnv(); err != nil {
		log.Fatalf("Unable to load environment settings: %v", err)
	}
	c.TestMode = c.Settings.DevMode
	log.Printf("%sRun() environment settings loaded", debugTag)

	// Initialize the database connection
	c.Db, err = InitDB(c.Settings.DataSource)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// Start background token cleanup job
	c.startTokenCleanupJob()
	// Start the email service
	debugEmail := ""
	if c.Settings.DevMode {
		debugEmail = c.Settings.EmailDebugAddr
	}
	log.Printf("%sRun() DEV_MODE=%t, email_debug_override_active=%t", debugTag, c.Settings.DevMode, debugEmail != "")
	c.EmailSvc, err = gmail.New(c.Settings.EmailSecret, c.Settings.EmailToken, c.Settings.EmailAddr, debugEmail, c.Settings.GmailAuthCode)
	if err != nil {
		log.Fatalf("Failed to initialize email gateway: %v", err)
	}

	// Start the payment service
	domain := "https://" + c.Settings.Host + ":" + c.Settings.PortHttps + c.Settings.APIprefix
	c.PaymentSvc = stripe.NewFromKey(c.Settings.PaymentKey, domain, c.TestMode)
	log.Printf(debugTag+"Run() Stripe gateway initialized with domain: %s", domain)
	if c.Settings.DevMode && c.Settings.StripeWebhookSecret == "" {
		log.Printf(debugTag + "Run() WARNING: DEV_MODE=true and STRIPE_WEBHOOK_SECRET is empty; webhook finalization is disabled unless webhook forwarding is configured. Dev fallback may apply.")
	}
	//go c.PaymentSvc.Start() // No longer needed as webhooks are not used.

	c.OAuthSvc, err = oAuthGateway.New(oAuthGateway.GatewayConfig{
		ClientID:       c.Settings.GoogleClientID,
		ClientSecret:   c.Settings.GoogleClientSecret,
		RedirectURL:    c.Settings.GoogleRedirectURL,
		SessionKey:     c.Settings.SessionKey,
		ClientRedirect: c.Settings.ClientRedirectURL,
		DevMode:        c.Settings.DevMode,
	})
	if err != nil {
		log.Fatalf("Failed to initialize oAuth Gateway: %v", err)
	}

}

func (c *Config) Close() {
	c.Db.Close()
}

// startTokenCleanupJob runs token cleanup every 15 minutes in the background
// This removes the performance overhead of cleaning tokens on every authenticated request
func (c *Config) startTokenCleanupJob() {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		// Run cleanup immediately on startup
		if err := dbAuthTemplate.TokenCleanExpired(debugTag+"tokenCleanup:", c.Db); err != nil {
			log.Printf("%sToken cleanup job failed on startup: %v", debugTag, err)
		} else {
			log.Printf("%sToken cleanup job completed successfully on startup", debugTag)
		}

		// Then run periodically
		for range ticker.C {
			if err := dbAuthTemplate.TokenCleanExpired(debugTag+"tokenCleanup:", c.Db); err != nil {
				log.Printf("%sToken cleanup job failed: %v", debugTag, err)
			} else {
				log.Printf("%sToken cleanup job completed successfully", debugTag)
			}
		}
	}()
	log.Printf("%sToken cleanup background job started (runs every 15 minutes)", debugTag)
}
