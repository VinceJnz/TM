package gmail

import (
	"context"
	"testing"
)

func TestDevelopmentMailRejectsInjectedRecipientHeader(t *testing.T) {
	gateway := &Gateway{from: "sender@example.test", debugEmailAddress: "tester@example.test"}
	sent, err := gateway.SendMailContext(context.Background(), "member@example.test",
		"Subject\r\nBcc: other@example.test", "Test message")
	if sent || err == nil {
		t.Fatal("accepted an extra recipient header")
	}
}
