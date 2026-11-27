package appearanceView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"context"
	"net/http"
	"strings"
	"syscall/js"
)

const ApiURL = "/siteAppearance"

type theme struct {
	Name       string `json:"name"`
	Primary    string `json:"primary"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Text       string `json:"text"`
	Font       string `json:"font"`
	Logo       string `json:"logo"`
	Icon       string `json:"icon"`
}

type settings struct {
	Version    int64  `json:"version"`
	Theme      theme  `json:"theme"`
	CanRestore bool   `json:"canRestore"`
	Action     string `json:"action,omitempty"`
}

type listener struct {
	node     js.Value
	event    string
	callback js.Func
}

type Editor struct {
	client                                                                             *httpProcessor.Client
	cancelLoad                                                                         context.CancelFunc
	doc, root, form, controls, feedback, preview, previewName, previewLogo             js.Value
	name, primary, background, surface, text, font, logo, icon, removeLogo, removeIcon js.Value
	listeners                                                                          []listener
	current                                                                            settings
	generation                                                                         int
}

func New(doc js.Value, app *appCore.AppCore) *Editor {
	e := &Editor{client: app.HttpClient, doc: doc}
	e.root = e.element(js.Undefined(), "div", "")
	return e
}

func (e *Editor) GetDiv() js.Value { return e.root }
func (e *Editor) Display()         { e.root.Set("hidden", false) }
func (e *Editor) Hide()            { e.root.Set("hidden", true); e.ResetView() }
func (e *Editor) ResetView() {
	e.generation++
	if e.cancelLoad != nil {
		e.cancelLoad()
		e.cancelLoad = nil
	}
	if !e.previewLogo.IsUndefined() && !e.previewLogo.IsNull() {
		revokePreview(e.previewLogo)
		e.previewLogo = js.Undefined()
	}
	for _, l := range e.listeners {
		l.node.Call("removeEventListener", l.event, l.callback)
		l.callback.Release()
	}
	e.listeners = nil
	e.root.Set("textContent", "")
}

func (e *Editor) element(parent js.Value, tag, label string) js.Value {
	el := e.doc.Call("createElement", tag)
	el.Set("textContent", label)
	if !parent.IsUndefined() {
		parent.Call("appendChild", el)
	}
	return el
}

func (e *Editor) on(node js.Value, event string, run func(js.Value)) {
	fn := js.FuncOf(func(_ js.Value, args []js.Value) any { run(args[0]); return nil })
	node.Call("addEventListener", event, fn)
	e.listeners = append(e.listeners, listener{node, event, fn})
}

func (e *Editor) button(parent js.Value, label string, run func()) js.Value {
	button := e.element(parent, "button", label)
	button.Set("type", "button")
	button.Set("className", "btn")
	e.on(button, "click", func(js.Value) { go run() })
	return button
}

func (e *Editor) field(parent js.Value, label, tag, kind, value string) js.Value {
	wrapper := e.element(parent, "label", label+" ")
	control := e.element(wrapper, tag, "")
	if kind != "" {
		control.Set("type", kind)
	}
	control.Set("value", value)
	return control
}

func (e *Editor) message(err error, fallback string) {
	if typed := httpProcessor.FindAPIError(err); typed != nil {
		fallback = typed.Message
	}
	e.feedback.Set("textContent", fallback)
}

func (e *Editor) FetchItems() {
	e.ResetView()
	generation := e.generation
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelLoad = cancel
	e.feedback = e.element(e.root, "p", "Loading appearance…")
	e.feedback.Set("role", "status")
	go func() {
		defer cancel()
		var current settings
		err := e.client.RequestContext(ctx, http.MethodGet, ApiURL, &current, nil)
		if generation != e.generation {
			return
		}
		if err != nil {
			e.message(err, "Appearance could not be loaded.")
			e.button(e.root, "Retry", e.FetchItems)
			return
		}
		e.current = current
		e.feedback.Set("textContent", "")
		e.render()
	}()
}

func (e *Editor) render() {
	t := e.current.Theme
	e.form = e.element(e.root, "form", "")
	e.form.Set("className", "tm-card")
	e.controls = e.element(e.form, "fieldset", "")
	e.element(e.controls, "legend", "Site branding")
	grid := e.element(e.controls, "div", "")
	grid.Set("className", "tm-appearance-fields")
	e.name = e.field(grid, "Site name", "input", "text", t.Name)
	e.name.Set("required", true)
	e.name.Set("maxLength", 80)
	e.primary = e.field(grid, "Brand color", "input", "color", t.Primary)
	e.background = e.field(grid, "Page background", "input", "color", t.Background)
	e.surface = e.field(grid, "Panel background", "input", "color", t.Surface)
	e.text = e.field(grid, "Text color", "input", "color", t.Text)
	e.font = e.field(grid, "Font", "select", "", "")
	for _, option := range [][2]string{{"system", "System"}, {"serif", "Serif"}, {"sans", "Sans serif"}} {
		el := e.element(e.font, "option", option[1])
		el.Set("value", option[0])
	}
	e.font.Set("value", t.Font)
	e.logo = e.field(grid, "Header logo (PNG or JPEG)", "input", "file", "")
	e.logo.Set("accept", "image/png,image/jpeg")
	e.icon = e.field(grid, "Browser icon (PNG or JPEG)", "input", "file", "")
	e.icon.Set("accept", "image/png,image/jpeg")
	e.removeLogo = e.field(grid, "Remove header logo", "input", "checkbox", "")
	e.removeIcon = e.field(grid, "Use default browser icon", "input", "checkbox", "")
	e.removeLogo.Set("disabled", t.Logo == "")
	e.removeIcon.Set("disabled", t.Icon == "")
	actions := e.element(e.controls, "div", "")
	actions.Set("className", "form-button-row")
	publish := e.element(actions, "button", "Publish appearance")
	publish.Set("type", "submit")
	publish.Set("className", "btn btn-primary")
	e.button(actions, "Restore defaults", func() { e.confirmPublish("defaults", "Publish the built-in appearance?") })
	if e.current.CanRestore {
		e.button(actions, "Restore previous", func() { e.confirmPublish("restore", "Publish the previous appearance?") })
	}
	e.button(actions, "Reload settings", func() {
		if !e.form.Call("hasAttribute", "data-admin-dirty").Bool() || viewHelpers.Confirm(e.root, "Discard unsaved appearance edits?") {
			e.FetchItems()
		}
	})
	e.element(e.root, "h3", "Preview")
	e.preview = e.element(e.root, "section", "")
	e.preview.Set("className", "tm-appearance-preview")
	brand := e.element(e.preview, "div", "")
	brand.Set("className", "tm-public-brand")
	e.previewLogo = e.element(brand, "img", "")
	e.previewLogo.Set("alt", "")
	e.previewName = e.element(brand, "span", "")
	panel := e.element(e.preview, "div", "")
	panel.Set("className", "tm-card")
	sample := e.element(panel, "label", "Example field ")
	example := e.element(sample, "input", "")
	example.Set("placeholder", "Sample text")
	button := e.element(panel, "button", "Example action")
	button.Set("className", "btn btn-primary")
	feedback := e.element(panel, "p", "Example status message")
	feedback.Set("className", "tm-feedback")
	e.on(e.form, "input", func(js.Value) { e.form.Call("setAttribute", "data-admin-dirty", ""); e.updatePreview() })
	e.on(e.form, "submit", func(event js.Value) { event.Call("preventDefault"); go e.publish("publish") })
	e.updatePreview()
}

func (e *Editor) updatePreview() {
	t := e.draft()
	style := e.preview.Get("style")
	style.Call("setProperty", "--tm-primary", t.Primary)
	style.Call("setProperty", "--tm-primary-hover", t.Primary)
	style.Call("setProperty", "--tm-bg", t.Background)
	style.Call("setProperty", "--tm-surface", t.Surface)
	style.Call("setProperty", "--tm-text", t.Text)
	fonts := map[string]string{"system": "system-ui, sans-serif", "serif": "Georgia, serif", "sans": "Arial, sans-serif"}
	style.Call("setProperty", "--tm-font", fonts[t.Font])
	e.previewName.Set("textContent", t.Name)
	src := "/site-logo"
	files := e.logo.Get("files")
	hasFile := files.Length() > 0
	visible := t.Logo != "" || hasFile
	if hasFile {
		src = js.Global().Get("URL").Call("createObjectURL", files.Index(0)).String()
	}
	revokePreview(e.previewLogo)
	if visible {
		e.previewLogo.Set("src", src)
	} else {
		e.previewLogo.Call("removeAttribute", "src")
	}
	e.previewLogo.Set("hidden", !visible)
}

func revokePreview(image js.Value) {
	old := image.Call("getAttribute", "src").String()
	if strings.HasPrefix(old, "blob:") {
		js.Global().Get("URL").Call("revokeObjectURL", old)
	}
}

func (e *Editor) draft() theme {
	t := e.current.Theme
	t.Name = e.name.Get("value").String()
	t.Primary = e.primary.Get("value").String()
	t.Background = e.background.Get("value").String()
	t.Surface = e.surface.Get("value").String()
	t.Text = e.text.Get("value").String()
	t.Font = e.font.Get("value").String()
	if e.removeLogo.Get("checked").Bool() {
		t.Logo = ""
	}
	if e.removeIcon.Get("checked").Bool() {
		t.Icon = ""
	}
	return t
}

func (e *Editor) confirmPublish(action, message string) {
	generation := e.generation
	if viewHelpers.Confirm(e.root, message) && generation == e.generation {
		e.publish(action)
	}
}

func (e *Editor) publish(action string) {
	if e.controls.Get("disabled").Bool() {
		return
	}
	e.controls.Set("disabled", true)
	generation := e.generation
	input := settings{Version: e.current.Version, Action: action}
	if action == "publish" {
		input.Theme = e.draft()
		for _, asset := range []struct {
			input js.Value
			max   int
			out   *string
			label string
		}{
			{e.logo, 256 * 1024, &input.Theme.Logo, "Header logo"}, {e.icon, 32 * 1024, &input.Theme.Icon, "Browser icon"},
		} {
			files := asset.input.Get("files")
			if files.Length() == 0 {
				continue
			}
			encoded, ok := readImage(files.Index(0), asset.max)
			if generation != e.generation {
				return
			}
			if !ok {
				e.feedback.Set("textContent", asset.label+" must be a PNG or JPEG within the upload limit.")
				e.controls.Set("disabled", false)
				return
			}
			*asset.out = encoded
		}
	}
	var published settings
	err := e.client.Request(http.MethodPut, ApiURL, &published, input)
	if generation != e.generation {
		return
	}
	e.controls.Set("disabled", false)
	if err != nil {
		e.message(err, "Appearance could not be published.")
		return
	}
	e.form.Call("removeAttribute", "data-admin-dirty")
	js.Global().Get("location").Call("reload")
}

func readImage(file js.Value, max int) (string, bool) {
	kind := file.Get("type").String()
	if file.Get("size").Int() > max || (kind != "image/png" && kind != "image/jpeg") {
		return "", false
	}
	reader := js.Global().Get("FileReader").New()
	result := make(chan string, 1)
	finished := js.FuncOf(func(js.Value, []js.Value) any {
		value := reader.Get("result")
		if value.Type() == js.TypeString {
			result <- value.String()
		} else {
			result <- ""
		}
		return nil
	})
	reader.Set("onloadend", finished)
	reader.Call("readAsDataURL", file)
	data := <-result
	reader.Set("onloadend", js.Null())
	finished.Release()
	_, encoded, ok := strings.Cut(data, ",")
	return encoded, ok && encoded != ""
}
