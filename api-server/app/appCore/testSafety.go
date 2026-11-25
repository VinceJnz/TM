package appCore

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
)

func flagValue(name string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}
	return v, nil
}
func (s *settings) EmailRecipientOverride() string {
	if s.DevMode {
		return s.EmailDevModeAddr
	}
	return ""
}
func validSingleEmail(value string) bool {
	a, err := mail.ParseAddress(value)
	return err == nil && a.Address == value && !strings.ContainsAny(value, "\r\n,;")
}
func (s *settings) loadTestSafety() error {
	var err error
	s.DevMode, err = flagValue("DEV_MODE", false)
	if err != nil {
		return err
	}
	s.EmailDevModeAddr = strings.TrimSpace(os.Getenv("EMAIL_DEV_MODE_ADDR"))
	s.StripeTestMode, err = flagValue("STRIPE_TEST_MODE", s.DevMode)
	if err != nil {
		return err
	}
	return s.validateTestSafety()
}
func (s *settings) validateTestSafety() error {
	if s.DevMode && !validSingleEmail(s.EmailDevModeAddr) {
		return fmt.Errorf("DEV_MODE requires one valid EMAIL_DEV_MODE_ADDR address")
	}
	if !s.DevMode && s.EmailDevModeAddr != "" {
		return fmt.Errorf("EMAIL_DEV_MODE_ADDR requires DEV_MODE=true")
	}
	if s.StripeTestMode && s.PaymentKey != "" && !strings.HasPrefix(s.PaymentKey, "sk_test_") && !strings.HasPrefix(s.PaymentKey, "rk_test_") {
		return fmt.Errorf("STRIPE_TEST_MODE requires a Stripe test secret/restricted key")
	}
	return nil
}

type redirectedMailer struct {
	base      EmailService
	recipient string
}

func (m redirectedMailer) RenewURL() string { return m.base.RenewURL() }
func (m redirectedMailer) SendMail(_ string, subject, body string) (bool, error) {
	return m.base.SendMail(m.recipient, subject, body)
}
func (m redirectedMailer) SendMailContext(ctx context.Context, _ string, subject, body string) (bool, error) {
	if contextual, ok := m.base.(interface {
		SendMailContext(context.Context, string, string, string) (bool, error)
	}); ok {
		return contextual.SendMailContext(ctx, m.recipient, subject, body)
	}
	return m.base.SendMail(m.recipient, subject, body)
}
