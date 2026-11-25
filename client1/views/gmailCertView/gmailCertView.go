package gmailCertView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"net/http"
	"strings"
	"syscall/js"
	"time"
)

const ApiURL = "/gmailcert"

type CertStatus struct {
	EmailAddr               string     `json:"email_addr"`
	TokenExists             bool       `json:"token_exists"`
	TokenModified           time.Time  `json:"token_modified"`
	SecretExists            bool       `json:"secret_exists"`
	SecretModified          time.Time  `json:"secret_modified"`
	GmailLoaded             bool       `json:"gmail_loaded"`
	GmailRefreshAvailable   bool       `json:"gmail_refresh_available"`
	GmailTokenExpiry        *time.Time `json:"gmail_token_expiry"`
	CanReconnect            bool       `json:"can_reconnect"`
	StripeLoaded            bool       `json:"stripe_loaded"`
	StripeWebhookConfigured bool       `json:"stripe_webhook_configured"`
	GoogleSignInLoaded      bool       `json:"google_sign_in_loaded"`
}

type listener struct {
	element  js.Value
	callback js.Func
}

type ItemEditor struct {
	app                              *appCore.AppCore
	client                           httpProcessor.Client
	document, div, content, feedback js.Value
	listeners                        []listener
	generation                       int
	busy                             bool
}

func New(document js.Value, _ *eventProcessor.EventProcessor, app *appCore.AppCore, _ ...int) *ItemEditor {
	editor := &ItemEditor{app: app, document: document, client: *app.HttpClient}
	editor.client.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	editor.div = editor.add(js.Undefined(), "div", "")
	editor.feedback = editor.add(editor.div, "p", "")
	editor.feedback.Call("setAttribute", "role", "status")
	editor.content = editor.add(editor.div, "div", "")
	return editor
}

func (e *ItemEditor) GetDiv() js.Value { return e.div }

func (e *ItemEditor) Display() { e.div.Get("style").Set("display", "block") }

func (e *ItemEditor) Hide() {
	e.div.Get("style").Set("display", "none")
	e.generation++
	e.clear()
	e.busy = false
}

func (e *ItemEditor) ResetView() {
	e.generation++
	e.clear()
	e.busy = false
}

func (e *ItemEditor) FetchItems() {
	if e.busy {
		return
	}
	if len(e.listeners) == 0 {
		e.addReloadButton()
	}
	e.load("Loading gateway status…", "")
}

func (e *ItemEditor) addReloadButton() {
	e.button(e.content, "Reload status", e.FetchItems)
}

func (e *ItemEditor) clear() {
	for _, listener := range e.listeners {
		listener.element.Call("removeEventListener", "click", listener.callback)
		listener.callback.Release()
	}
	e.listeners = nil
	e.content.Set("textContent", "")
}

func (e *ItemEditor) add(parent js.Value, tag, label string) js.Value {
	element := e.document.Call("createElement", tag)
	element.Set("textContent", label)
	if !parent.IsUndefined() {
		parent.Call("appendChild", element)
	}
	return element
}

func (e *ItemEditor) button(parent js.Value, label string, action func()) {
	button := e.add(parent, "button", label)
	button.Set("type", "button")
	viewHelpers.StyleButtonPrimary(button)
	callback := js.FuncOf(func(this js.Value, args []js.Value) any {
		if !e.busy {
			action()
		}
		return nil
	})
	button.Call("addEventListener", "click", callback)
	e.listeners = append(e.listeners, listener{button, callback})
}

func (e *ItemEditor) request(method, path, activity string, result, payload any, done func()) {
	e.busy = true
	e.feedback.Set("textContent", activity)
	generation := e.generation
	go e.client.NewRequest2(method, path, result, payload, httpProcessor.Callbacks{
		OnSuccess: func() {
			if generation == e.generation {
				e.busy = false
				done()
			}
		},
		OnError: func(err error) {
			if generation == e.generation {
				e.busy = false
				message := "Request failed. Check your connection and permissions, then retry."
				if _, body, found := strings.Cut(err.Error(), "server message="); found && strings.TrimSpace(body) != "" {
					message = strings.TrimSpace(body)
				}
				e.feedback.Set("textContent", message)
			}
		},
	})
}

func (e *ItemEditor) load(activity, message string) {
	var status CertStatus
	e.request(http.MethodGet, ApiURL, activity, &status, nil, func() {
		e.clear()
		e.render(status)
		e.feedback.Set("textContent", message)
	})
}

func (e *ItemEditor) render(status CertStatus) {
	e.addReloadButton()
	gmail := e.providerPanel("Gmail")
	e.add(gmail, "p", "Sender: "+status.EmailAddr)
	e.add(gmail, "p", "Client: "+configured(status.GmailLoaded))
	e.add(gmail, "p", fileStatus("Token file", status.TokenExists, status.TokenModified))
	e.add(gmail, "p", fileStatus("OAuth client file", status.SecretExists, status.SecretModified))
	if status.GmailTokenExpiry != nil {
		e.add(gmail, "p", "Saved access token expires: "+status.GmailTokenExpiry.UTC().Format("2006-01-02 15:04 UTC"))
	}

	if e.app.HasAccess("gmailcert", []string{"put"}, []string{"any"}) {
		if status.GmailRefreshAvailable {
			e.button(gmail, "Refresh Gmail token", func() {
				e.update(ApiURL+"/refresh", nil)
			})
		}
		if status.CanReconnect {
			link := e.add(gmail, "p", "")
			e.button(gmail, "Reconnect Gmail", func() {
				var result struct {
					URL     string `json:"url"`
					Message string `json:"message"`
				}
				link.Set("textContent", "")
				e.request(http.MethodGet, ApiURL+"/renew-url", "Preparing Gmail authorization…", &result, nil, func() {
					e.feedback.Set("textContent", result.Message)
					authorize := e.add(link, "a", "Continue to Google")
					authorize.Set("href", result.URL)
					authorize.Set("target", "_blank")
					authorize.Set("rel", "noopener noreferrer")
				})
			})
		} else {
			e.add(gmail, "p", "To enable reconnection, configure the Gmail credential paths and register the GMAIL_RENEW_REDIRECT_URL callback in Google Cloud.")
		}
		advanced := e.add(gmail, "details", "")
		e.add(advanced, "summary", "Replace Gmail credential files")
		e.upload(advanced, "OAuth client JSON", "secret")
		e.upload(advanced, "Token JSON", "token")
	}

	stripe := e.providerPanel("Stripe")
	e.add(stripe, "p", "API client: "+configured(status.StripeLoaded))
	e.add(stripe, "p", "Webhook signing secret: "+configured(status.StripeWebhookConfigured))
	e.add(stripe, "p", "Rotate credentials in Stripe Dashboard, update PAYMENT_KEY or STRIPE_WEBHOOK_SECRET in deployment settings, and restart the API.")
	google := e.providerPanel("Google sign-in")
	e.add(google, "p", "OAuth client: "+configured(status.GoogleSignInLoaded))
	e.add(google, "p", "Rotate the OAuth client secret in Google Cloud, update GOOGLE_CLIENT_SECRET in deployment settings, and restart the API.")
}

func (e *ItemEditor) providerPanel(title string) js.Value {
	panel := e.add(e.content, "section", "")
	panel.Set("className", "tm-card tm-admin-provider")
	e.add(panel, "h3", title)
	return panel
}

func (e *ItemEditor) upload(parent js.Value, title, kind string) {
	label := e.add(parent, "label", title+" ")
	input := e.add(label, "input", "")
	input.Set("type", "password")
	input.Set("autocomplete", "off")
	e.button(parent, "Save "+title, func() {
		content := strings.TrimSpace(input.Get("value").String())
		if content == "" {
			e.feedback.Set("textContent", "Paste the credential JSON before saving.")
			return
		}
		input.Set("value", "")
		e.update(ApiURL, map[string]string{"type": kind, "content": content})
	})
}

func (e *ItemEditor) update(path string, payload any) {
	var result struct {
		Message string `json:"message"`
	}
	e.request(http.MethodPut, path, "Updating Gmail credentials…", &result, payload, func() {
		e.load("Reloading status…", result.Message)
	})
}

func configured(ok bool) string {
	if ok {
		return "Configured"
	}
	return "Unavailable"
}

func fileStatus(name string, exists bool, modified time.Time) string {
	if !exists {
		return name + ": missing"
	}
	return name + ": updated " + modified.UTC().Format("2006-01-02 15:04 UTC")
}
