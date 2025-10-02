package handlerWebAuthn

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/webAuthnPool"

	"github.com/go-webauthn/webauthn/webauthn"
)

const debugTag = "handlerWebAuthn."

const WebAuthnSessionTokenName = "_temp_session_token"
const WebAuthnEmailTokenName = "_temp_email_token"

type Handler struct {
	appConf  *appCore.Config
	webAuthn *webauthn.WebAuthn // webAuthn instance for handling WebAuthn operations
	Pool     *webAuthnPool.Pool // Uncomment if you want to use a pool for session data
}

func New(appConf *appCore.Config) *Handler {
	wconfig := &webauthn.Config{
		RPDisplayName: appConf.Settings.AppTitle,                                                       // Display name for your app
		RPID:          appConf.Settings.Host,                                                           // Your domain (no https://, just domain)
		RPOrigins:     []string{"https://" + appConf.Settings.Host + ":" + appConf.Settings.PortHttps}, // Full origin URLs
		// Correct timeout configuration for newer library versions
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Timeout:    60000, // 60 seconds in milliseconds
				Enforce:    true,
				TimeoutUVD: 60000, // User verification device timeout
			},
			Registration: webauthn.TimeoutConfig{
				Timeout:    60000, // 60 seconds in milliseconds
				Enforce:    true,
				TimeoutUVD: 60000, // User verification device timeout
			},
		},
	}

	webAuthnInstance, err := webauthn.New(wconfig)
	if err != nil {
		panic("failed to create WebAuthn from config: " + err.Error())
	}

	return &Handler{
		webAuthn: webAuthnInstance,
		appConf:  appConf,
		Pool:     webAuthnPool.New(),
	}
}
