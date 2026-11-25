package appCore

import (
	"context"
	"testing"
)

type capturedMailer struct{ to string }

func (m *capturedMailer) SendMail(to, subject, body string) (bool, error) {
	m.to = to
	return true, nil
}
func (m *capturedMailer) RenewURL() string { return "" }
func TestDevelopmentMailUsesOnlyConfiguredInbox(t *testing.T) {
	t.Setenv("DEV_MODE", "true")
	t.Setenv("EMAIL_DEV_MODE_ADDR", "tester@example.test")
	t.Setenv("STRIPE_TEST_MODE", "true")
	s := settings{PaymentKey: "sk_test_synthetic"}
	if err := s.loadTestSafety(); err != nil {
		t.Fatal(err)
	}
	app := New(false)
	app.Settings = s
	m := &capturedMailer{}
	app.SetEmailService(m)
	app.EmailService().SendMail("registered@example.test", "synthetic", "synthetic")
	if m.to != "tester@example.test" {
		t.Fatal("registered address used")
	}
	app.EmailService().(interface {
		SendMailContext(context.Context, string, string, string) (bool, error)
	}).SendMailContext(context.Background(), "another@example.test", "synthetic", "synthetic")
	if m.to != "tester@example.test" {
		t.Fatal("context mail escaped redirect")
	}
	replacement := &capturedMailer{}
	app.SetEmailService(replacement)
	app.EmailService().SendMail("registered@example.test", "synthetic", "synthetic")
	if replacement.to != "tester@example.test" {
		t.Fatal("renewed mailer escaped redirect")
	}
	for _, recipient := range []string{"", "a@example.test,b@example.test", "a@example.test\r\nBcc: real@example.test"} {
		t.Setenv("EMAIL_DEV_MODE_ADDR", recipient)
		if err := s.loadTestSafety(); err == nil {
			t.Fatal("unsafe redirect accepted")
		}
		app.Settings = s
		if app.EmailService() != nil {
			t.Fatal("mailer available with invalid development recipient")
		}
	}
	t.Setenv("EMAIL_DEV_MODE_ADDR", "tester@example.test")
	s.PaymentKey = "sk_live_synthetic"
	if err := s.loadTestSafety(); err == nil {
		t.Fatal("live Stripe key accepted")
	}
	t.Setenv("STRIPE_TEST_MODE", "invalid")
	if err := s.loadTestSafety(); err == nil {
		t.Fatal("invalid flag accepted")
	}
}

func TestProductionMailUsesRequestedInbox(t *testing.T) {
	t.Setenv("DEV_MODE", "false")
	t.Setenv("STRIPE_TEST_MODE", "false")
	t.Setenv("EMAIL_DEV_MODE_ADDR", "tester@example.test")
	app := New(false)
	if err := app.Settings.loadTestSafety(); err == nil {
		t.Fatal("development recipient accepted outside development mode")
	}
	t.Setenv("EMAIL_DEV_MODE_ADDR", "")
	if err := app.Settings.loadTestSafety(); err != nil {
		t.Fatal(err)
	}
	m := &capturedMailer{}
	app.SetEmailService(m)
	app.EmailService().SendMail("registered@example.test", "synthetic", "synthetic")
	if m.to != "registered@example.test" {
		t.Fatal("production recipient was overridden")
	}
}
