package domEvents

import "syscall/js"

type listener struct {
	target   js.Value
	event    string
	callback js.Func
}

var listeners = map[int]listener{}
var sequence int
var observer js.Value
var observerCallback js.Func

// Timeout releases its callback after execution or explicit cancellation.
func Timeout(callback func(js.Value, []js.Value) any, milliseconds int) func() {
	active := true
	var handler js.Func
	handler = js.FuncOf(func(this js.Value, args []js.Value) any {
		if !active {
			return nil
		}
		active = false
		defer handler.Release()
		return callback(this, args)
	})
	id := js.Global().Call("setTimeout", handler, milliseconds)
	return func() {
		if active {
			active = false
			js.Global().Call("clearTimeout", id)
			handler.Release()
		}
	}
}

// Then and Catch release both branches when the promise settles, including the
// branch that never executes. Rejections keep propagating through the chain.
func Then(promise js.Value, callback func(js.Value, []js.Value) any) js.Value {
	return settle(promise, callback, false)
}
func Catch(promise js.Value, callback func(js.Value, []js.Value) any) js.Value {
	return settle(promise, callback, true)
}
func settle(promise js.Value, callback func(js.Value, []js.Value) any, catch bool) js.Value {
	var success, failure js.Func
	release := func() { success.Release(); failure.Release() }
	success = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer release()
		if !catch {
			return callback(this, args)
		}
		if len(args) > 0 {
			return args[0]
		}
		return nil
	})
	failure = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer release()
		if catch {
			return callback(this, args)
		}
		if len(args) > 0 {
			return js.Global().Get("Promise").Call("reject", args[0])
		}
		return js.Global().Get("Promise").Call("reject", "promise rejected")
	})
	return promise.Call("then", success, failure)
}

// Listen owns a DOM listener until its target subtree is removed. The observer
// is shared for the lifetime of the application, rather than one per row/button.
func Listen(target js.Value, event string, callback func(js.Value, []js.Value) any) {
	ensureObserver()
	sequence++
	id := sequence
	handler := js.FuncOf(callback)
	listeners[id] = listener{target, event, handler}
	target.Call("addEventListener", event, handler)
}

// ReleaseTree can also be called explicitly by an owning view at destruction.
func ReleaseTree(root js.Value) {
	for id, item := range listeners {
		if root.Equal(item.target) || root.Call("contains", item.target).Bool() {
			item.target.Call("removeEventListener", item.event, item.callback)
			item.callback.Release()
			delete(listeners, id)
		}
	}
}
func ensureObserver() {
	if !observer.IsUndefined() && !observer.IsNull() {
		return
	}
	constructor := js.Global().Get("MutationObserver")
	document := js.Global().Get("document")
	if constructor.IsUndefined() || document.IsUndefined() {
		return
	}
	observerCallback = js.FuncOf(func(_ js.Value, args []js.Value) any {
		records := args[0]
		for i := 0; i < records.Length(); i++ {
			nodes := records.Index(i).Get("removedNodes")
			for j := 0; j < nodes.Length(); j++ {
				node := nodes.Index(j)
				// Moving a node within the document does not destroy its owner.
				if !node.Get("isConnected").Bool() {
					ReleaseTree(node)
				}
			}
		}
		return nil
	})
	observer = constructor.New(observerCallback)
	observer.Call("observe", document.Get("documentElement"), map[string]any{"childList": true, "subtree": true})
}
