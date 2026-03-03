package contentView

import (
	"client1/v2/app/appCore"
	"net/http"
	"strings"
	"syscall/js"
)

const ApiURL = "/auth"

type Section struct {
	Heading    string
	Paragraphs []string
}

type Page struct {
	Title string `json:"title"`
	HTML  string `json:"html"`
}

type View struct {
	appCore       *appCore.AppCore
	document      js.Value
	div           js.Value
	pageKey       string
	fallbackTitle string
	loaded        bool
	isLoading     bool
}

func New(document js.Value, appCore *appCore.AppCore, pageKey, fallbackTitle string) *View {
	v := &View{}
	v.appCore = appCore
	v.document = document
	v.pageKey = strings.ToLower(strings.TrimSpace(pageKey))
	v.fallbackTitle = strings.TrimSpace(fallbackTitle)
	v.div = document.Call("createElement", "div")
	v.div.Set("id", "contentView"+slugify(v.pageKey))
	v.div.Set("className", "content-view")
	v.div.Get("style").Call("setProperty", "display", "none")
	v.renderLoading()
	return v
}

func (v *View) Display() {
	v.div.Get("style").Call("setProperty", "display", "block")
}

func (v *View) FetchItems() {
	if v == nil || v.appCore == nil || v.appCore.HttpClient == nil {
		v.renderError("content service unavailable")
		return
	}
	if v.loaded || v.isLoading {
		return
	}

	v.isLoading = true
	v.renderLoading()

	var page Page
	success := func(err error) {
		v.isLoading = false
		if err != nil {
			v.renderError(err.Error())
			return
		}
		v.loaded = true
		v.renderPage(page)
	}
	fail := func(err error) {
		v.isLoading = false
		if err == nil {
			v.renderError("failed to load content")
			return
		}
		v.renderError(err.Error())
	}

	go func() {
		v.appCore.HttpClient.NewRequest(http.MethodGet, ApiURL+"/content/"+v.pageKey+"/", &page, nil, success, fail)
	}()
}

func (v *View) Hide() {
	v.div.Get("style").Call("setProperty", "display", "none")
}

func (v *View) GetDiv() js.Value {
	return v.div
}

func (v *View) ResetView() {
	v.loaded = false
	v.isLoading = false
	v.renderLoading()
}

func (v *View) renderPage(page Page) {
	html := strings.TrimSpace(page.HTML)
	if html == "" {
		v.renderError("empty page content")
		return
	}
	v.div.Set("innerHTML", html)
}

func (v *View) renderLoading() {
	v.div.Set("innerHTML", "")
	title := v.document.Call("createElement", "h2")
	title.Set("textContent", v.fallbackTitle)
	v.div.Call("appendChild", title)

	message := v.document.Call("createElement", "p")
	message.Set("textContent", "Loading content...")
	v.div.Call("appendChild", message)
}

func (v *View) renderError(message string) {
	v.div.Set("innerHTML", "")
	title := v.document.Call("createElement", "h2")
	title.Set("textContent", v.fallbackTitle)
	v.div.Call("appendChild", title)

	errorText := v.document.Call("createElement", "p")
	errorText.Set("textContent", "Unable to load content from server.")
	v.div.Call("appendChild", errorText)

	if strings.TrimSpace(message) != "" {
		detail := v.document.Call("createElement", "p")
		detail.Set("textContent", message)
		v.div.Call("appendChild", detail)
	}
}

func slugify(input string) string {
	value := strings.ToLower(strings.TrimSpace(input))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "--", "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "page"
	}
	return value
}
