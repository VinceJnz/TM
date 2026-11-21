package oAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"strings"
	"syscall/js"
	"testing"
	"time"
)

func TestOAuthRestorationUsesConfiguredPrefixAndRequiresEnsure(t *testing.T) {
	for _, mode := range []string{"session", "oauth", "ensure-failure"} {
		t.Run(mode, func(t *testing.T) {
			old := js.Global().Get("fetch")
			defer js.Global().Set("fetch", old)
			js.Global().Set("__tmOAuthCalls", js.ValueOf([]any{}))
			defer js.Global().Delete("__tmOAuthCalls")
			js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `
   globalThis.__tmOAuthCalls.push(url);
   let status=200,body={name:'Synthetic',email:'member@example.test'};
   if(url.endsWith('/menuUser/')) {status='`+mode+`'==='session'?200:401;body={user_id:1,name:'Synthetic'};}
   if(url.endsWith('/ensure')&&'`+mode+`'==='ensure-failure') status=503;
   return Promise.resolve(new Response(JSON.stringify(body),{status}));
  `))
			events := eventProcessor.New()
			finished := make(chan bool, 4)
			events.AddEventHandler(eventProcessor.EventTypeLoginComplete, func(eventProcessor.Event) { finished <- true })
			events.AddEventHandler(eventProcessor.EventTypeDisplayMessage, func(event eventProcessor.Event) {
				if message, ok := event.Data.(string); ok && strings.Contains(message, "Could not establish") {
					finished <- false
				}
			})
			editor := &ItemEditor{client: httpProcessor.New("https://example.test/custom/v9"), events: events}
			editor.authProcess()
			select {
			case loggedIn := <-finished:
				if loggedIn != (mode != "ensure-failure") || editor.LoggedIn != loggedIn {
					t.Fatal(loggedIn, editor.LoggedIn)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("authentication remained blocked")
			}
			calls := js.Global().Get("__tmOAuthCalls")
			want := 1
			if mode != "session" {
				want = 3
			}
			if calls.Length() != want {
				t.Fatal(calls.Length(), want)
			}
			for i := 0; i < calls.Length(); i++ {
				if !strings.HasPrefix(calls.Index(i).String(), "https://example.test/custom/v9/auth/") {
					t.Fatal(calls.Index(i).String())
				}
			}
		})
	}
}
