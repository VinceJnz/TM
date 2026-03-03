package handlerAuth

import (
	"strings"
	"testing"
)

func TestExtractMainHTML(t *testing.T) {
	doc := `<!doctype html><html><body><main class="main"><h1>Home</h1><p>Hello</p></main></body></html>`
	got, ok := extractMainHTML(doc)
	if !ok {
		t.Fatalf("expected main content to be found")
	}
	if !strings.Contains(got, "<main") || !strings.Contains(got, "<h1>Home</h1>") {
		t.Fatalf("unexpected extracted main content: %s", got)
	}

	_, ok = extractMainHTML(`<html><body><div>No main</div></body></html>`)
	if ok {
		t.Fatalf("expected no main content")
	}
}

func TestSanitizeHTML_RemovesDangerousContent(t *testing.T) {
	in := `<main onclick="evil()"><h1>Title</h1><script>alert(1)</script><a href="javascript:alert(1)" onclick="x()">link</a><iframe src="https://bad"></iframe></main>`
	out := sanitizeHTML(in)

	if strings.Contains(strings.ToLower(out), "<script") {
		t.Fatalf("expected script tag removed, got: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "onclick=") {
		t.Fatalf("expected inline event handlers removed, got: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "<iframe") {
		t.Fatalf("expected iframe removed, got: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "javascript:") {
		t.Fatalf("expected javascript href removed, got: %s", out)
	}
	if !strings.Contains(out, `href="#"`) {
		t.Fatalf("expected unsafe href rewritten to #, got: %s", out)
	}
}

func TestSanitizeHTML_TargetBlankAddsRel(t *testing.T) {
	in := `<main><a href="https://example.com" target="_blank">open</a></main>`
	out := sanitizeHTML(in)
	if !strings.Contains(out, `target="_blank"`) {
		t.Fatalf("expected target to be preserved, got: %s", out)
	}
	if !strings.Contains(out, `rel="noopener noreferrer"`) {
		t.Fatalf("expected rel noopener noreferrer to be added, got: %s", out)
	}
}

func TestRewriteInternalLinksToSPA(t *testing.T) {
	in := `<main><a href="/about.html">About</a> <a href="contact.html">Contact</a> <a href="/index.html">Home</a></main>`
	out := rewriteInternalLinksToSPA(in)

	if !strings.Contains(out, `href="#/about"`) {
		t.Fatalf("expected about link rewritten, got: %s", out)
	}
	if !strings.Contains(out, `href="#/contact"`) {
		t.Fatalf("expected contact link rewritten, got: %s", out)
	}
	if !strings.Contains(out, `href="#/"`) {
		t.Fatalf("expected home link rewritten, got: %s", out)
	}
}
