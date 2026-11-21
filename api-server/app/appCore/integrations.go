package appCore

import (
	"fmt"
	"strings"
)

type IntegrationHealth struct {
	Available bool `json:"available"`
	Required  bool `json:"required"`
}

func parseRequiredIntegrations(value string) ([]string, error) {
	names := []string{}
	seen := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		switch name {
		case "mail", "stripe", "google":
		default:
			return nil, fmt.Errorf("unknown REQUIRED_INTEGRATIONS entry")
		}
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	return names, nil
}

// These are local initialization/configuration signals, not provider probes.
func (c *Config) IntegrationStatus() map[string]IntegrationHealth {
	status := map[string]IntegrationHealth{
		"mail":   {Available: c.EmailService() != nil},
		"stripe": {Available: c.PaymentSvc != nil && c.PaymentSvc.Client != nil && c.Settings.StripeWebhookSecret != ""},
		"google": {Available: c.OAuthSvc != nil},
	}
	for _, name := range c.Settings.RequiredIntegrations {
		v := status[name]
		v.Required = true
		status[name] = v
	}
	return status
}
func (c *Config) IntegrationAlerts() []string {
	status := c.IntegrationStatus()
	alerts := []string{}
	for _, name := range []string{"mail", "stripe", "google"} {
		v := status[name]
		if v.Required && !v.Available {
			alerts = append(alerts, "required_"+name+"_unavailable")
		}
	}
	return alerts
}
