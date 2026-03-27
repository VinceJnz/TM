package gmailCertView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"log"
	"net/http"
	"syscall/js"
	"time"
)

const debugTag = "gmailCertView."
const ApiURL = "/gmailcert"

type ItemState int

const (
	ItemStateNone ItemState = iota
	ItemStateFetching
	ItemStateSaving
)

type ViewState int

const (
	ViewStateNone  ViewState = iota
	ViewStateBlock ViewState = iota
)

// CertStatus mirrors the server's GET response.
type CertStatus struct {
	EmailAddr      string    `json:"email_addr"`
	TokenFile      string    `json:"token_file"`
	TokenExists    bool      `json:"token_exists"`
	TokenModified  time.Time `json:"token_modified"`
	SecretFile     string    `json:"secret_file"`
	SecretExists   bool      `json:"secret_exists"`
	SecretModified time.Time `json:"secret_modified"`
}

// UpdateRequest mirrors the server's PUT request body.
type UpdateRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// RenewURLResponse mirrors the server's GET /renew-url response.
type RenewURLResponse struct {
	URL     string `json:"url"`
	Message string `json:"message"`
}

type UI struct {
	TokenTextarea  js.Value
	SecretTextarea js.Value
	RenewURLDiv    js.Value
}

type ItemEditor struct {
	appCore   *appCore.AppCore
	client    *httpProcessor.Client
	document  js.Value
	events    *eventProcessor.EventProcessor
	ItemState ItemState
	ViewState ViewState
	Status    CertStatus
	UI        UI
	Div       js.Value
	StatusDiv js.Value
	EditDiv   js.Value
	StateDiv  js.Value
}

func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := &ItemEditor{
		appCore:  appCore,
		document: document,
		events:   events,
		client:   appCore.HttpClient,
	}

	editor.Div = document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	editor.StatusDiv = document.Call("createElement", "div")
	editor.StatusDiv.Set("id", debugTag+"StatusDiv")
	editor.Div.Call("appendChild", editor.StatusDiv)

	editor.EditDiv = document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"EditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	editor.StateDiv = document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"StateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	return editor
}

func (editor *ItemEditor) GetDiv() js.Value { return editor.Div }

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

func (editor *ItemEditor) Hide() {
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) ResetView() {
	editor.StatusDiv.Set("innerHTML", "")
	editor.EditDiv.Set("innerHTML", "")
	editor.FetchItems()
}

func (editor *ItemEditor) FetchItems() {
	go func() {
		editor.setStateText("Loading...")
		var status CertStatus
		editor.client.NewRequest(http.MethodGet, ApiURL, &status, nil)
		editor.Status = status
		editor.renderView()
		editor.setStateText("")
	}()
}

// renderView builds the status display and the two upload forms.
func (editor *ItemEditor) renderView() {
	editor.StatusDiv.Set("innerHTML", "")
	editor.EditDiv.Set("innerHTML", "")

	// --- Status section ---
	statusCard := editor.document.Call("createElement", "div")
	viewHelpers.SetStyles(statusCard, map[string]string{
		"border":        "1px solid #ccc",
		"border-radius": "6px",
		"padding":       "12px 16px",
		"margin-bottom": "16px",
		"background":    "#f9f9f9",
	})

	heading := editor.document.Call("createElement", "h3")
	heading.Set("innerText", "Gmail Gateway Status")
	viewHelpers.SetStyleProperty(heading, "margin-top", "0")
	statusCard.Call("appendChild", heading)

	statusCard.Call("appendChild", editor.infoRow("From address", editor.Status.EmailAddr))
	statusCard.Call("appendChild", editor.fileStatusRow("Token file", editor.Status.TokenFile, editor.Status.TokenExists, editor.Status.TokenModified))
	statusCard.Call("appendChild", editor.fileStatusRow("Secret file", editor.Status.SecretFile, editor.Status.SecretExists, editor.Status.SecretModified))

	editor.StatusDiv.Call("appendChild", statusCard)

	// --- Renew URL section ---
	editor.EditDiv.Call("appendChild", editor.buildRenewURLSection())

	// --- Token upload form ---
	editor.EditDiv.Call("appendChild", editor.buildUploadSection(
		"Update Token File",
		"Paste the contents of client_token.json below.",
		"token",
		&editor.UI.TokenTextarea,
		"tokenContent",
		editor.submitTokenUpdate,
	))

	// --- Secret upload form ---
	editor.EditDiv.Call("appendChild", editor.buildUploadSection(
		"Update Secret File",
		"Paste the contents of client_secret.json below.",
		"secret",
		&editor.UI.SecretTextarea,
		"secretContent",
		editor.submitSecretUpdate,
	))
}

func (editor *ItemEditor) infoRow(label, value string) js.Value {
	row := editor.document.Call("createElement", "p")
	row.Set("innerHTML", "<strong>"+label+":</strong> "+value)
	viewHelpers.SetStyleProperty(row, "margin", "4px 0")
	return row
}

func (editor *ItemEditor) fileStatusRow(label, path string, exists bool, modified time.Time) js.Value {
	status := "missing"
	modStr := ""
	if exists {
		status = "present"
		modStr = " — last modified " + modified.Format("2006-01-02 15:04:05 UTC")
	}
	row := editor.document.Call("createElement", "p")
	row.Set("innerHTML", "<strong>"+label+":</strong> <code>"+path+"</code> ("+status+modStr+")")
	viewHelpers.SetStyleProperty(row, "margin", "4px 0")
	return row
}

func (editor *ItemEditor) buildUploadSection(title, hint, certType string, textareaPtr *js.Value, textareaID string, onSubmit func(js.Value, []js.Value) interface{}) js.Value {
	card := editor.document.Call("createElement", "div")
	viewHelpers.SetStyles(card, map[string]string{
		"border":        "1px solid #ccc",
		"border-radius": "6px",
		"padding":       "12px 16px",
		"margin-bottom": "16px",
	})

	h4 := editor.document.Call("createElement", "h4")
	h4.Set("innerText", title)
	viewHelpers.SetStyleProperty(h4, "margin-top", "0")
	card.Call("appendChild", h4)

	hintEl := editor.document.Call("createElement", "p")
	hintEl.Set("innerText", hint)
	viewHelpers.SetStyles(hintEl, map[string]string{"color": "#555", "margin": "0 0 8px"})
	card.Call("appendChild", hintEl)

	textarea := editor.document.Call("createElement", "textarea")
	textarea.Set("id", textareaID)
	textarea.Set("placeholder", "{}")
	textarea.Set("rows", "8")
	viewHelpers.SetStyles(textarea, map[string]string{
		"width":         "100%",
		"font-family":   "monospace",
		"font-size":     "0.85em",
		"box-sizing":    "border-box",
		"margin-bottom": "8px",
	})
	*textareaPtr = textarea
	card.Call("appendChild", textarea)

	saveBtn := viewHelpers.Button(onSubmit, editor.document, "Save "+title, certType+"SaveBtn")
	viewHelpers.StyleButtonPrimary(saveBtn)
	card.Call("appendChild", saveBtn)

	return card
}

func (editor *ItemEditor) buildRenewURLSection() js.Value {
	card := editor.document.Call("createElement", "div")
	viewHelpers.SetStyles(card, map[string]string{
		"border":        "1px solid #ccc",
		"border-radius": "6px",
		"padding":       "12px 16px",
		"margin-bottom": "16px",
	})

	h4 := editor.document.Call("createElement", "h4")
	h4.Set("innerText", "Get OAuth Renewal URL")
	viewHelpers.SetStyleProperty(h4, "margin-top", "0")
	card.Call("appendChild", h4)

	hintEl := editor.document.Call("createElement", "p")
	hintEl.Set("innerText", "Click the button to generate a Google authorisation URL. Open it in a browser, grant access, then paste the returned auth code into the token update form below.")
	viewHelpers.SetStyles(hintEl, map[string]string{"color": "#555", "margin": "0 0 8px"})
	card.Call("appendChild", hintEl)

	getBtn := viewHelpers.Button(editor.fetchRenewURL, editor.document, "Get Renewal URL", "getRenewURLBtn")
	viewHelpers.StyleButtonPrimary(getBtn)
	card.Call("appendChild", getBtn)

	// Container where the URL will be displayed after fetch.
	urlDiv := editor.document.Call("createElement", "div")
	urlDiv.Set("id", "renewURLResult")
	viewHelpers.SetStyleProperty(urlDiv, "margin-top", "10px")
	editor.UI.RenewURLDiv = urlDiv
	card.Call("appendChild", urlDiv)

	return card
}

func (editor *ItemEditor) fetchRenewURL(this js.Value, p []js.Value) interface{} {
	go func() {
		editor.setStateText("Fetching renewal URL...")
		var resp RenewURLResponse
		editor.client.NewRequest(http.MethodGet, ApiURL+"/renew-url", &resp, nil)
		editor.setStateText("")
		if resp.URL == "" {
			editor.UI.RenewURLDiv.Set("innerHTML", "<p style='color:red'>Failed to retrieve renewal URL.</p>")
			return
		}
		editor.UI.RenewURLDiv.Set("innerHTML",
			"<p style='margin:4px 0'>"+resp.Message+"</p>"+
				"<a href='"+resp.URL+"' target='_blank' rel='noopener noreferrer' "+
				"style='word-break:break-all;font-size:0.85em'>"+resp.URL+"</a>")
	}()
	return nil
}

func (editor *ItemEditor) submitTokenUpdate(this js.Value, p []js.Value) interface{} {
	go editor.submitUpdate("token", editor.UI.TokenTextarea)
	return nil
}

func (editor *ItemEditor) submitSecretUpdate(this js.Value, p []js.Value) interface{} {
	go editor.submitUpdate("secret", editor.UI.SecretTextarea)
	return nil
}

func (editor *ItemEditor) submitUpdate(certType string, textarea js.Value) {
	content := textarea.Get("value").String()
	if content == "" {
		editor.onCompletionMsg("Content is empty — nothing was saved.")
		return
	}

	// Validate JSON client-side before sending.
	if !json.Valid([]byte(content)) {
		editor.onCompletionMsg("The pasted content is not valid JSON. Please check and try again.")
		return
	}

	editor.setStateText("Saving...")
	req := UpdateRequest{Type: certType, Content: content}
	editor.client.NewRequest(http.MethodPut, ApiURL, nil, &req)
	editor.setStateText("")
	editor.onCompletionMsg(certType + " cert saved successfully.")
	log.Printf("%ssubmitUpdate %s cert submitted", debugTag, certType)
	// Refresh status display to show updated modification time.
	editor.FetchItems()
}

func (editor *ItemEditor) setStateText(msg string) {
	editor.StateDiv.Set("innerText", msg)
}

func (editor *ItemEditor) onCompletionMsg(msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{
		Type:     eventProcessor.EventTypeDisplayMessage,
		DebugTag: debugTag,
		Data:     msg,
	})
}
