package appCore

import (
	"api-server/v2/app/gateways/gmail"
	"api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	"api-server/v2/app/gateways/stripe"
	"api-server/v2/app/services/checkout"
	"api-server/v2/app/services/notifications"
	"api-server/v2/migrations"
	notificationStore "api-server/v2/modelMethods/notifications"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

const debugTag = "appCore."

type ContextKey string

func GenerateSessionIDContextKey() ContextKey { // User for generating the context key for passing values via the context (ctx)
	return ContextKey("sessionID")
}

type EmailService interface {
	SendMail(string, string, string) (bool, error)
	RenewURL() string
}

type Config struct {
	Db               *sqlx.DB
	EmailSvc         EmailService
	emailMu          sync.RWMutex
	PaymentSvc       *stripe.Gateway
	OAuthSvc         *oAuthGateway.Gateway
	SessionIDKey     ContextKey // User for passing the user id value via the context (ctx)
	Settings         settings
	TestMode         bool
	stop             context.CancelFunc
	cleanupDone      chan struct{}
	paymentStop      context.CancelFunc
	paymentDone      chan struct{}
	notificationStop context.CancelFunc
	notificationDone chan struct{}
}

func New(testMode bool) *Config {
	sessionIDKey := GenerateSessionIDContextKey()
	return &Config{

		SessionIDKey: sessionIDKey,
		TestMode:     testMode,
	}
}

func (c *Config) Run(ctx context.Context) error {
	var err error
	// Load environment variables
	if err = c.Settings.LoadEnv(); err != nil {
		return err
	}
	c.TestMode = c.Settings.DevMode
	log.Printf("%sRun() environment settings loaded", debugTag)

	// Initialize the database connection
	c.Db, err = InitDBContext(ctx, c.Settings.DataSource)
	if err != nil {
		return errors.New("database initialization failed")
	}

	// Start background token cleanup job
	if err := migrations.Check(ctx, c.Db); err != nil {
		return err
	}
	c.startTokenCleanupJob(ctx)
	// Start the email service
	debugEmail := c.Settings.EmailRecipientOverride()
	log.Printf("%sRun() DEV_MODE=%t, email_debug_override_active=%t", debugTag, c.Settings.DevMode, debugEmail != "")
	c.EmailSvc, err = gmail.New(c.Settings.EmailSecret, c.Settings.EmailToken, c.Settings.EmailAddr, debugEmail, c.Settings.GmailAuthCode)
	if err != nil {
		c.EmailSvc = nil
		log.Print("email integration unavailable")
	}

	notificationCtx, notificationCancel := context.WithCancel(ctx)
	c.notificationStop = notificationCancel
	c.notificationDone = make(chan struct{})
	go func() {
		defer close(c.notificationDone)
		(notifications.Service{ReviewGroups: strings.Split(c.Settings.AdminNotifyGroup, ","), DeliveredRetentionDays: c.Settings.NotificationDeliveredRetentionDays, OrphanRetentionDays: c.Settings.NotificationOrphanRetentionDays, Store: notificationStore.Store{DB: c.Db}, Mailer: func() notifications.Mailer { return c.EmailService() }}).Run(notificationCtx)
	}()
	// Start the payment service
	domain := "https://" + c.Settings.Host + ":" + c.Settings.PortHttps + c.Settings.APIprefix
	c.PaymentSvc = stripe.NewFromKey(c.Settings.PaymentKey, domain, c.Settings.StripeTestMode)
	c.PaymentSvc.RecipientOverride = c.Settings.EmailRecipientOverride()
	if c.Settings.PaymentKey != "" {
		paymentCtx, cancel := context.WithCancel(ctx)
		c.paymentStop = cancel
		c.paymentDone = make(chan struct{})
		go func() { defer close(c.paymentDone); checkout.New(c.Db, c.PaymentSvc).RunRecovery(paymentCtx) }()
	}
	log.Printf(debugTag+"Run() Stripe gateway initialized with domain: %s", domain)
	if c.Settings.DevMode && c.Settings.StripeWebhookSecret == "" {
		log.Printf(debugTag + "Run() WARNING: DEV_MODE=true and STRIPE_WEBHOOK_SECRET is empty; webhook finalization is disabled unless webhook forwarding is configured. Configure webhook forwarding for local tests.")
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
		c.OAuthSvc = nil
		log.Print("OAuth integration unavailable")
	}

	return nil
}

func (c *Config) EmailService() EmailService {
	c.emailMu.RLock()
	defer c.emailMu.RUnlock()
	if c.EmailSvc == nil {
		return nil
	}
	if recipient := c.Settings.EmailRecipientOverride(); recipient != "" {
		return redirectedMailer{base: c.EmailSvc, recipient: recipient}
	}
	return c.EmailSvc
}

func (c *Config) SetEmailService(service EmailService) {
	c.emailMu.Lock()
	defer c.emailMu.Unlock()
	c.EmailSvc = service
}

func (c *Config) Close() {
	if c.notificationStop != nil {
		c.notificationStop()
	}
	if c.notificationDone != nil {
		<-c.notificationDone
	}
	if c.paymentStop != nil {
		c.paymentStop()
	}
	if c.paymentDone != nil {
		<-c.paymentDone
	}
	if c.stop != nil {
		c.stop()
	}
	if c.cleanupDone != nil {
		<-c.cleanupDone
	}
	if c.Db != nil {
		c.Db.Close()
	}
}

// Exactly one cleanup job per application, stopped before closing the database.
func (c *Config) startTokenCleanupJob(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	c.stop = cancel
	c.cleanupDone = make(chan struct{})
	go func() {
		defer close(c.cleanupDone)
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		cleanup := func() {
			job, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if _, err := c.Db.ExecContext(job, "DELETE FROM st_token WHERE valid_to<CURRENT_TIMESTAMP"); err != nil && ctx.Err() == nil {
				log.Print("token cleanup failed")
			}
		}
		cleanup()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()
}
