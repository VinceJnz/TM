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
	if s.EmailRedirectAll {
		return s.EmailRedirectTo
	}
	if !s.emailRedirectConfigured && s.DevMode {
		return s.EmailDebugAddr
	}
	return ""
}
func validSingleEmail(value string) bool {
	a, err := mail.ParseAddress(value)
	return err == nil && a.Address == value && !strings.ContainsAny(value, "\r\n,;")
}
func (s *settings) loadTestSafety() error {
	var err error
	s.emailRedirectConfigured = strings.TrimSpace(os.Getenv("EMAIL_REDIRECT_ALL")) != ""
	s.EmailRedirectAll, err = flagValue("EMAIL_REDIRECT_ALL", s.DevMode && s.EmailDebugAddr != "")
	if err != nil {
		return err
	}
	s.EmailRedirectTo = strings.TrimSpace(os.Getenv("EMAIL_REDIRECT_TO"))
	if s.EmailRedirectTo == "" && s.EmailRedirectAll {
		s.EmailRedirectTo = strings.TrimSpace(s.EmailDebugAddr)
	}
	s.StripeTestMode, err = flagValue("STRIPE_TEST_MODE", s.DevMode)
	if err != nil {
		return err
	}
	return s.validateTestSafety()
}
func (s *settings) validateTestSafety() error {
	if s.EmailRedirectAll && !validSingleEmail(s.EmailRedirectTo) {
		return fmt.Errorf("EMAIL_REDIRECT_ALL requires one valid EMAIL_REDIRECT_TO address")
	}
	if !s.EmailRedirectAll && s.EmailRedirectTo != "" {
		return fmt.Errorf("EMAIL_REDIRECT_TO requires EMAIL_REDIRECT_ALL=true")
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
