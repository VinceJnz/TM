//go:build js && wasm

package httpProcessor

import (
	"io"
	"net/http"
	"strings"
	"syscall/js"
	"testing"
	"time"
)

func TestFetchHandlesBodyRejectionAndSuccess(t *testing.T) {
	for _, reject := range []bool{true, false} {
		oldFetch := js.Global().Get("fetch")
		text := js.FuncOf(func(js.Value, []js.Value) any {
			if reject {
				return js.Global().Get("Promise").Call("reject", js.Global().Get("Error").New("synthetic body failure"))
			}
			return js.Global().Get("Promise").Call("resolve", "complete body")
		})
		fetch := js.FuncOf(func(js.Value, []js.Value) any {
			return js.Global().Get("Promise").Call("resolve", js.ValueOf(map[string]any{"status": 200, "statusText": "OK", "text": text}))
		})
		js.Global().Set("fetch", fetch)
		client := New("https://example.test")
		request, _ := http.NewRequest("GET", "https://example.test/", nil)
		finished := make(chan error, 1)
		go func() {
			response, err := client.doRequest(request)
			if err == nil {
				data, readErr := io.ReadAll(response.Body)
				response.Body.Close()
				if string(data) != "complete body" {
					t.Error("response body was lost")
				}
				err = readErr
			}
			finished <- err
		}()
		select {
		case err := <-finished:
			if reject && (err == nil || !strings.Contains(err.Error(), "synthetic body failure")) {
				t.Fatalf("body rejection did not reach caller: %v", err)
			}
			if !reject && err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("body rejection left request blocked")
		}
		js.Global().Set("fetch", oldFetch)
		fetch.Release()
		text.Release()
	}
}

func TestListFollowsNextOffsetWithoutLosingRows(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `
  const next = new URL(url).searchParams.get('offset');
  return Promise.resolve(new Response(JSON.stringify(next ? [{id:2}] : [{id:1}]),
    {status:200, headers:next ? {} : {'X-Next-Offset':'1'}}));
 `))
	client := New("https://example.test/custom/v9")
	var records []struct {
		ID int `json:"id"`
	}
	finished := make(chan error, 1)
	go func() { finished <- client.newRequest("GET", "/bookings", &records, nil) }()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("paginated fetch blocked")
	}
	if len(records) != 2 || records[0].ID != 1 || records[1].ID != 2 {
		t.Fatal(records)
	}
}
