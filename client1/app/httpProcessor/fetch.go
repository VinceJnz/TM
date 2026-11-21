package httpProcessor

import (
	"client1/v2/app/domEvents"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall/js"
)

type fetchResult struct {
	response *http.Response
	err      error
}

func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	ctx, finish, err := c.lifecycle.begin(req.Context())
	if err != nil {
		return nil, err
	}
	defer finish()
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if c.HTTPClient != nil && c.HTTPClient.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.HTTPClient.Timeout)
		defer cancel()
	}
	constructor := js.Global().Get("AbortController")
	if constructor.IsUndefined() || constructor.IsNull() {
		return nil, fmt.Errorf("browser AbortController unavailable")
	}
	controller := constructor.New()
	options := map[string]any{"method": req.Method, "credentials": "include", "mode": "cors", "signal": controller.Get("signal")}
	headers := map[string]any{}
	for name, values := range req.Header {
		if len(values) > 0 {
			headers[name] = values[0]
		}
	}
	options["headers"] = headers
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		options["body"] = string(body)
	}
	results := make(chan fetchResult, 1)
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			controller.Call("abort")
		case <-finished:
		}
	}()
	promise := js.Global().Call("fetch", req.URL.String(), options)
	chain := domEvents.Then(promise, func(_ js.Value, args []js.Value) any {
		response := args[0]
		result := &http.Response{StatusCode: response.Get("status").Int(), Status: response.Get("statusText").String(), Header: make(http.Header)}
		headerJS := response.Get("headers")
		if !headerJS.IsUndefined() && !headerJS.IsNull() {
			callback := js.FuncOf(func(_ js.Value, values []js.Value) any {
				result.Header.Set(values[1].String(), values[0].String())
				return nil
			})
			headerJS.Call("forEach", callback)
			callback.Release()
		}
		return domEvents.Then(response.Call("text"), func(_ js.Value, values []js.Value) any {
			body := ""
			if len(values) > 0 {
				body = values[0].String()
			}
			result.Body = &stringReadCloser{strings.NewReader(body)}
			results <- fetchResult{response: result}
			return nil
		})
	})
	domEvents.Catch(chain, func(_ js.Value, args []js.Value) any {
		message := "fetch failed"
		if len(args) > 0 && !args[0].IsNull() && !args[0].IsUndefined() {
			if value := args[0].Get("message"); value.Truthy() {
				message = value.String()
			}
		}
		results <- fetchResult{err: fmt.Errorf("fetch failed: %s", message)}
		return nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-results:
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return result.response, result.err
	}
}
