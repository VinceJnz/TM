//go:build js && wasm

package httpProcessor

import (
	"context"
	"errors"
	"syscall/js"
	"testing"
	"time"
)

func TestCancellationAndDestroyAbortFetchAndUnblockWaiters(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `globalThis.__tmStarted=true;return new Promise((resolve,reject)=>options.signal.addEventListener('abort',()=>{globalThis.__tmAborted=true;reject(new Error('cancelled'));},{once:true}));`))
	defer js.Global().Delete("__tmStarted")
	defer js.Global().Delete("__tmAborted")
	for _, destroy := range []bool{false, true} {
		js.Global().Set("__tmStarted", false)
		js.Global().Set("__tmAborted", false)
		c := New("https://example.test")
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- c.RequestContext(ctx, "GET", "/bookings", nil, nil) }()
		for i := 0; i < 100 && !js.Global().Get("__tmStarted").Bool(); i++ {
			time.Sleep(time.Millisecond)
		}
		if destroy {
			c.Destroy()
		} else {
			cancel()
		}
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("cancel left blocked waiter")
		}
		for i := 0; i < 100 && !js.Global().Get("__tmAborted").Bool(); i++ {
			time.Sleep(time.Millisecond)
		}
		if !js.Global().Get("__tmAborted").Bool() {
			t.Fatal("fetch not aborted")
		}
		if destroy && !errors.Is(c.Request("GET", "/bookings", nil, nil), context.Canceled) {
			t.Fatal("destroyed client reused")
		}
		cancel()
	}
}
