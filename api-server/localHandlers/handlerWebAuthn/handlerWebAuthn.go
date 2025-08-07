package handlerWebAuthn

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/webAuthnPool"

	"github.com/go-webauthn/webauthn/webauthn"
)

const debugTag = "handlerWebAuthn."

const WebAuthnSessionCookieName = "_temp_session"

type Handler struct {
	appConf  *appCore.Config
	webAuthn *webauthn.WebAuthn // webAuthn instance for handling WebAuthn operations
	Pool     *webAuthnPool.Pool // Uncomment if you want to use a pool for session data
}

func New(appConf *appCore.Config) *Handler {
	webAuthnInstance, err := webauthn.New(&webauthn.Config{
		RPDisplayName: appConf.Settings.AppTitle,
		RPID:          appConf.Settings.Host,
		RPOrigins:     []string{"https://" + appConf.Settings.Host + ":" + appConf.Settings.PortHttps},
	})
	if err != nil {
		panic("failed to create WebAuthn from config: " + err.Error())
	}

	return &Handler{
		webAuthn: webAuthnInstance,
		appConf:  appConf,
		Pool:     webAuthnPool.New(),
	}
}
