//go:build js && wasm

package httpProcessor

import (
	"bytes"
	"log"
	"strings"
	"syscall/js"
	"testing"
)

func TestDecodeFailureDiagnosticsExcludePayloadAndURL(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `return Promise.resolve(new Response('{"name":"private-person","id":"private-token"}',{status:200,headers:{"X-Request-ID":"0123456789abcdef01234567"}}));`))
	var logs bytes.Buffer
	output := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(output)
	var result struct {
		Name string
		ID   int
	}
	err := New("https://example.test").Request("GET", "/auth?token=private-query", &result, nil)
	if err == nil {
		t.Fatal("bad response accepted")
	}
	for _, secret := range []string{"private-person", "private-token", "private-query"} {
		if strings.Contains(logs.String(), secret) || strings.Contains(err.Error(), secret) {
			t.Fatal("sensitive diagnostics", secret)
		}
	}
	if !strings.Contains(logs.String(), "invalid_json") || !strings.Contains(logs.String(), "0123456789abcdef01234567") {
		t.Fatal(logs.String())
	}
}
