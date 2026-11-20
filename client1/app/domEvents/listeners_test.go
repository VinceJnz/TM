package domEvents

import (
	"syscall/js"
	"testing"
	"time"
)

func TestObserverReleasesDetachedSubtree(t *testing.T) {
	oldDocument := js.Global().Get("document")
	oldConstructor := js.Global().Get("MutationObserver")
	defer func() {
		observerCallback.Release()
		observer = js.Undefined()
		js.Global().Set("document", oldDocument)
		js.Global().Set("MutationObserver", oldConstructor)
		js.Global().Delete("__tmObserver")
	}()
	js.Global().Set("document", js.ValueOf(map[string]any{"documentElement": map[string]any{}}))
	js.Global().Set("MutationObserver", js.Global().Get("Function").New("callback", `globalThis.__tmObserver=callback;this.observe=function(){};`))
	target := js.Global().Get("EventTarget").New()
	target.Set("isConnected", false)
	called := false
	Listen(target, "click", func(js.Value, []js.Value) any { called = true; return nil })
	js.Global().Get("__tmObserver").Invoke(js.ValueOf([]any{map[string]any{"removedNodes": []any{target}}}))
	target.Call("dispatchEvent", js.Global().Get("Event").New("click"))
	if called {
		t.Fatal("removed subtree retained its listener")
	}
}

func TestPromiseBranchesPreserveSuccessAndFailure(t *testing.T) {
	completed := make(chan string, 2)
	success := js.Global().Get("Promise").Call("resolve", "value")
	Catch(Then(success, func(_ js.Value, args []js.Value) any { completed <- args[0].String(); return nil }), func(js.Value, []js.Value) any { completed <- "unexpected rejection"; return nil })
	failure := js.Global().Get("Promise").Call("reject", "failure")
	Catch(Then(failure, func(js.Value, []js.Value) any { completed <- "unexpected success"; return nil }), func(_ js.Value, args []js.Value) any { completed <- args[0].String(); return nil })
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case value := <-completed:
			seen[value] = true
		case <-time.After(time.Second):
			t.Fatal("promise did not settle")
		}
	}
	if !seen["value"] || !seen["failure"] {
		t.Fatal(seen)
	}
}

func TestRemovedOwnerReleasesListeners(t *testing.T) {
	baseline := len(listeners)
	for i := 0; i < 100; i++ {
		target := js.Global().Get("EventTarget").New()
		calls := 0
		Listen(target, "click", func(js.Value, []js.Value) any { calls++; return nil })
		target.Call("dispatchEvent", js.Global().Get("Event").New("click"))
		if calls != 1 {
			t.Fatal("listener did not run")
		}
		ReleaseTree(target)
		ReleaseTree(target)
		target.Call("dispatchEvent", js.Global().Get("Event").New("click"))
		if calls != 1 || len(listeners) != baseline {
			t.Fatal("destroyed view retained a callback")
		}
	}
}

func TestTimeoutExecutionAndOwnerCancellation(t *testing.T) {
	done := make(chan struct{}, 1)
	cancel := Timeout(func(js.Value, []js.Value) any { done <- struct{}{}; return nil }, 1)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timer did not execute")
	}
	cancel()
	cancel()
	cancelled := false
	cancel = Timeout(func(js.Value, []js.Value) any { cancelled = true; return nil }, 5)
	cancel()
	cancel()
	<-time.After(20 * time.Millisecond)
	if cancelled {
		t.Fatal("destroyed owner's timer ran")
	}
}
