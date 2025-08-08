package webAuthnResetView

import (
	"encoding/json"
	"errors"
	"syscall/js"
)

// API endpoints (adjust as needed)
const (
	BeginResetURL  = "https://localhost:8086/api/v1/auth/webauthn/email-reset"
	FinishResetURL = "https://localhost:8086/api/v1/auth/webauthn/email-reset/finish"
)

// To start the process (request email reset):
//editor.WebAuthnEmailReset(TableData{Email: "user@example.com"}, "")

// To finish the process (after user clicks the email link and you have the token):
//editor.WebAuthnEmailReset(TableData{}, "token-from-url")

// Step 1: Request reset link by email
func BeginWebAuthnEmailReset(email string, onResult func(success bool, message string)) {
	if email == "" {
		onResult(false, "Email is required")
		return
	}

	go func() {
		// Prepare request body
		req := map[string]string{"email": email}
		body, err := json.Marshal(req)
		if err != nil {
			onResult(false, "Failed to encode request")
			return
		}

		// Prepare fetch options
		opts := map[string]interface{}{
			"method": "POST",
			"body":   string(body),
			"headers": map[string]interface{}{
				"Content-Type": "application/json",
			},
		}

		// Call fetch
		promise := js.Global().Call("fetch", BeginResetURL, js.ValueOf(opts))
		resp, err := await(promise)
		if err != nil {
			onResult(false, "Network error: "+err.Error())
			return
		}
		if !resp.Get("ok").Bool() {
			msg := resp.Get("statusText").String()
			onResult(false, "Server error: "+msg)
			return
		}

		// Parse JSON response
		jsonPromise := resp.Call("json")
		result, err := await(jsonPromise)
		if err != nil {
			onResult(false, "Invalid server response")
			return
		}

		success := result.Get("success").Bool()
		message := result.Get("message").String()
		onResult(success, message)
	}()
}

// Step 2: Finish reset using token from email link
func FinishWebAuthnEmailReset(token string, onResult func(success bool, message string, credentialsRemoved int, nextSteps []string)) {
	if token == "" {
		onResult(false, "Reset token is required", 0, nil)
		return
	}

	go func() {
		url := FinishResetURL + "?token=" + token
		promise := js.Global().Call("fetch", url)
		resp, err := await(promise)
		if err != nil {
			onResult(false, "Network error: "+err.Error(), 0, nil)
			return
		}
		if !resp.Get("ok").Bool() {
			msg := resp.Get("statusText").String()
			onResult(false, "Server error: "+msg, 0, nil)
			return
		}

		jsonPromise := resp.Call("json")
		result, err := await(jsonPromise)
		if err != nil {
			onResult(false, "Invalid server response", 0, nil)
			return
		}

		success := result.Get("success").Bool()
		message := result.Get("message").String()
		credentialsRemoved := 0
		if result.Get("credentials_removed").Type() == js.TypeNumber {
			credentialsRemoved = result.Get("credentials_removed").Int()
		}
		var nextSteps []string
		if arr := result.Get("next_steps"); arr.Type() == js.TypeObject && arr.Length() > 0 {
			for i := 0; i < arr.Length(); i++ {
				nextSteps = append(nextSteps, arr.Index(i).String())
			}
		}
		onResult(success, message, credentialsRemoved, nextSteps)
	}()
}

// Helper: Await a JS promise in TinyGo
func await(promise js.Value) (js.Value, error) {
	ch := make(chan struct {
		val js.Value
		err error
	}, 1)
	then := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ch <- struct {
			val js.Value
			err error
		}{args[0], nil}
		return nil
	})
	catch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ch <- struct {
			val js.Value
			err error
		}{js.Null(), errors.New(args[0].String())}
		return nil
	})
	promise.Call("then", then).Call("catch", catch)
	result := <-ch
	then.Release()
	catch.Release()
	return result.val, result.err
}
