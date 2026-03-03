package handlerAuth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/net/html"
)

type contentPage struct {
	Title string `json:"title"`
	HTML  string `json:"html"`
}

var mainTagRegex = regexp.MustCompile(`(?is)<main[^>]*>.*?</main>`)

var allowedHTMLTags = map[string]bool{
	"main":    true,
	"section": true,
	"article": true,
	"header":  true,
	"h1":      true,
	"h2":      true,
	"h3":      true,
	"h4":      true,
	"h5":      true,
	"h6":      true,
	"p":       true,
	"ul":      true,
	"ol":      true,
	"li":      true,
	"a":       true,
	"strong":  true,
	"em":      true,
	"b":       true,
	"i":       true,
	"u":       true,
	"span":    true,
	"div":     true,
	"br":      true,
}

var dropNodeAndChildrenTags = map[string]bool{
	"script":   true,
	"style":    true,
	"iframe":   true,
	"object":   true,
	"embed":    true,
	"noscript": true,
}

var allowedGlobalAttrs = map[string]bool{
	"class": true,
	"id":    true,
	"title": true,
	"role":  true,
}

type contentSource struct {
	Title    string
	FileName string
}

var allowedContentPages = map[string]contentSource{
	"home": {
		Title:    "Home",
		FileName: "home.html",
	},
	"about": {
		Title:    "About",
		FileName: "about.html",
	},
	"contact": {
		Title:    "Contact",
		FileName: "contact.html",
	},
}

func (h *Handler) ContentGet(w http.ResponseWriter, r *http.Request) {
	pageKey := strings.ToLower(strings.TrimSpace(mux.Vars(r)["page"]))
	if pageKey == "" {
		pageKey = "home"
	}

	source, ok := allowedContentPages[pageKey]
	if !ok {
		writeJSONResponse(w, http.StatusNotFound, map[string]string{
			"error": "page not found",
		}, "ContentGet()")
		return
	}

	pagePath := filepath.Join("static", source.FileName)
	body, err := os.ReadFile(pagePath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONResponse(w, http.StatusNotFound, map[string]string{
				"error": "page not found",
			}, "ContentGet()")
			return
		}
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read page content",
		}, "ContentGet()")
		return
	}

	html, ok := extractMainHTML(string(body))
	if !ok {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "page missing <main> content",
		}, "ContentGet()")
		return
	}

	html = sanitizeHTML(html)
	html = rewriteInternalLinksToSPA(html)
	if strings.TrimSpace(html) == "" {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "empty page content",
		}, "ContentGet()")
		return
	}

	etag := contentETag(pageKey, html)
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("ETag", etag)

	payload := contentPage{
		Title: source.Title,
		HTML:  html,
	}

	writeJSONResponse(w, http.StatusOK, payload, "ContentGet()")
}

func extractMainHTML(doc string) (string, bool) {
	match := mainTagRegex.FindString(doc)
	if strings.TrimSpace(match) == "" {
		return "", false
	}
	return strings.TrimSpace(match), true
}

func sanitizeHTML(rawHTML string) string {
	wrapped := "<!doctype html><html><body>" + rawHTML + "</body></html>"
	doc, err := html.Parse(strings.NewReader(wrapped))
	if err != nil {
		return ""
	}
	body := findFirstElement(doc, "body")
	if body == nil {
		return ""
	}

	root := &html.Node{Type: html.ElementNode, Data: "div"}
	for node := body.FirstChild; node != nil; node = node.NextSibling {
		appendSanitizedNode(root, node)
	}

	var b strings.Builder
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&b, child); err != nil {
			return ""
		}
	}

	return strings.TrimSpace(b.String())
}

func findFirstElement(node *html.Node, tag string) *html.Node {
	if node == nil {
		return nil
	}
	if node.Type == html.ElementNode && strings.EqualFold(node.Data, tag) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirstElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func appendSanitizedNode(parent *html.Node, node *html.Node) {
	if node == nil {
		return
	}

	switch node.Type {
	case html.TextNode:
		if strings.TrimSpace(node.Data) == "" && parent.Data != "pre" {
			return
		}
		parent.AppendChild(&html.Node{Type: html.TextNode, Data: node.Data})
	case html.ElementNode:
		tag := strings.ToLower(strings.TrimSpace(node.Data))
		if dropNodeAndChildrenTags[tag] {
			return
		}

		if allowedHTMLTags[tag] {
			sanitized := &html.Node{Type: html.ElementNode, Data: tag}
			sanitized.Attr = sanitizeAttributes(tag, node.Attr)
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				appendSanitizedNode(sanitized, child)
			}
			parent.AppendChild(sanitized)
			return
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			appendSanitizedNode(parent, child)
		}
	case html.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			appendSanitizedNode(parent, child)
		}
	default:
		return
	}
}

func sanitizeAttributes(tag string, attrs []html.Attribute) []html.Attribute {
	result := make([]html.Attribute, 0, len(attrs))
	hasRel := false
	hasTargetBlank := false

	for _, attr := range attrs {
		key := strings.ToLower(strings.TrimSpace(attr.Key))
		val := strings.TrimSpace(attr.Val)

		if strings.HasPrefix(key, "on") {
			continue
		}

		switch key {
		case "href":
			if tag != "a" {
				continue
			}
			result = append(result, html.Attribute{Key: "href", Val: sanitizeHref(val)})
		case "target":
			if tag != "a" {
				continue
			}
			target := strings.ToLower(val)
			if target == "_blank" {
				hasTargetBlank = true
			}
			if target == "" {
				continue
			}
			result = append(result, html.Attribute{Key: "target", Val: target})
		case "rel":
			if tag != "a" {
				continue
			}
			hasRel = true
			if val == "" {
				continue
			}
			result = append(result, html.Attribute{Key: "rel", Val: val})
		default:
			if allowedGlobalAttrs[key] {
				result = append(result, html.Attribute{Key: key, Val: val})
			}
		}
	}

	if tag == "a" && hasTargetBlank && !hasRel {
		result = append(result, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
	}

	return result
}

func sanitizeHref(raw string) string {
	href := strings.TrimSpace(raw)
	if href == "" {
		return "#"
	}

	lower := strings.ToLower(href)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return "#"
	}

	if strings.HasPrefix(href, "#") ||
		strings.HasPrefix(href, "/") ||
		strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") {
		return href
	}

	return "#"
}

func rewriteInternalLinksToSPA(html string) string {
	replacer := strings.NewReplacer(
		`href="/about.html"`, `href="#/about"`,
		`href="about.html"`, `href="#/about"`,
		`href="/contact.html"`, `href="#/contact"`,
		`href="contact.html"`, `href="#/contact"`,
		`href="/home.html"`, `href="#/"`,
		`href="home.html"`, `href="#/"`,
		`href="/index.html"`, `href="#/"`,
		`href="index.html"`, `href="#/"`,
		`href='\/about.html'`, `href='#/about'`,
		`href='about.html'`, `href='#/about'`,
		`href='\/contact.html'`, `href='#/contact'`,
		`href='contact.html'`, `href='#/contact'`,
		`href='\/home.html'`, `href='#/'`,
		`href='home.html'`, `href='#/'`,
		`href='\/index.html'`, `href='#/'`,
		`href='index.html'`, `href='#/'`,
	)
	return replacer.Replace(html)
}

func contentETag(pageKey, html string) string {
	sum := sha256.Sum256([]byte(pageKey + ":" + html))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}
