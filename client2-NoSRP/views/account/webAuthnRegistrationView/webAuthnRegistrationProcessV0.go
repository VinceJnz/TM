package webAuthnRegistrationView

import (
	"encoding/json"
	"log"
	"syscall/js"
	"time"
)

//**********************************************************************************************************************
//**********************************************************************************************************************
//**********************************************************************************************************************
//**********************************************************************************************************************

// WebAuthnRegistration handles the registration process for WebAuthn
// 1. Send user data to server to get registration options.
// 2. Convert challenge and user ID to the correct format for WebAuthn.
// 3. Call browser WebAuthn API to create credentials.
// 4. Send credentials back to server to finish registration.
func (editor *ItemEditor) WebAuthnRegistrationV0(userData TableData) {

	registrationFinished := func(this js.Value, args []js.Value) any {
		log.Printf(debugTag + "registrationFinished()1 Registration process finished.")
		// You can add any UI updates or notifications here to inform the user that registration is complete.
		// For example, you might want to display a success message or redirect the user to another page.
		return nil
	}

	// Handle the result of the credentials creation. This function is called when the credentials are created. (then2)
	// It shows the token dialog to get the emailed token from the user.
	// It sends the credentials to the server to finish the registration.
	sendCredentials := func(this js.Value, args []js.Value) any {
		log.Printf(debugTag + "sendCredentials()1 Created credentials using WebAuthn API.")
		cred := args[0]

		processToken := func(token string) {
			if token == "" {
				log.Printf("Registration cancelled: No token provided.")
				return
			}
			credJSON := js.Global().Get("JSON").Call("stringify", cred)
			js.Global().Get("fetch").Invoke(ApiURL+"/register/finish/"+token, map[string]any{
				"method": "POST",
				"body":   credJSON,
				"headers": map[string]any{
					"Content-Type": "application/json",
				},
			})
			registrationFinished(js.Undefined(), nil) // Call registrationFinished after sending credentials
		}

		cancelProcess := func() {
			log.Printf("Registration cancelled by user.")
		}

		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any { // ????
			ShowTokenDialog(
				processToken,
				cancelProcess,
			)
			return nil
		}), 10) // ????
		return nil
	}

	// This function prepares the publicKey options for WebAuthn credentials creation (then1).
	prepCredentials := func(this js.Value, args []js.Value) any {
		log.Printf(debugTag+"prepCredentials()1 Parsed JSON response for registration options. userData: %+v", userData)
		options := args[0]
		publicKey := options.Get("publicKey")

		// Convert challenge and user.id from base64url to Uint8Array, the correct format for WebAuthn
		displayName := userData.Name + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
		publicKey.Set("challenge", decodeBase64URLToUint8Array("prepCredentials()2", publicKey.Get("challenge").String()))
		user := publicKey.Get("user")
		user.Set("id", decodeBase64URLToUint8Array("prepCredentials()3", user.Get("id").String()))
		user.Set("displayName", displayName) // <-- Add this line to set the nickname. If this is provided, browser shows it in UI and the existing credential is updated.
		publicKey.Set("user", user)

		log.Printf(debugTag+"prepCredentials()4 Converted Convert challenge and user.id from base64url to Uint8Array: displayName: %+v, user: %+v", displayName, user.String())

		// Call the browser WebAuthn API to create credentials
		credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
			"publicKey": publicKey,
		})

		log.Printf(debugTag+"prepCredentials()5 Called WebAuthn API to create credentials. credPromise: %+v", credPromise)
		credPromise.Call("then", sendCredentials)
		return nil
	}

	// 1. Send user data to server to get registration options.
	sendBeginRequest := func() any {
		// Function that receives the response promise from fetch. This receives the options from the server.
		fetch := js.Global().Get("fetch")
		log.Printf(debugTag+"sendBeginRequest()0 Starting WebAuthn registration process. userData: %+v", userData)
		// Marshal editor.CurrentRecord to JSON
		userDataJSON, err := json.Marshal(userData)
		if err != nil {
			log.Printf(debugTag+"WebAuthnRegistration()1 Error marshalling user data to JSON: %v", err)
			// Handle error
			return js.Undefined()
		}
		// Prepare user data as JSON string for the request body
		userDataStr := string(userDataJSON)
		log.Printf(debugTag+"sendBeginRequest()2 Preparing to send registration begin request to server. userData: %+v", userDataStr)
		// Send POST request to /register/begin/ to get registration options
		respPromiseOptions := fetch.Invoke(ApiURL+"/register/begin/", map[string]any{
			"method": "POST",
			"body":   userDataStr,
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
		})
		respPromiseOptions.Call("then", prepCredentials)
		return nil
	}

	log.Printf(debugTag + "WebAuthnRegistration()2 Starting WebAuthn registration process.")
	sendBeginRequest()

	//getOptions := sendBeginRequest()
	//log.Printf(debugTag+"WebAuthnRegistration()3 Sent registration begin request to server. getOptions: %+v", getOptions)
	//getCredentials := getOptions.Call("then", prepCredentials)
	//log.Printf(debugTag+"WebAuthnRegistration()4 Prepared credentials. getCredentials: %+v", getCredentials)
	//finishRegistration := getCredentials.Call("then", sendCredentials)
	//log.Printf(debugTag+"WebAuthnRegistration()5 Sent credentials to server to finish registration. finishRegistration: %+v", finishRegistration)
	//finishRegistration.Call("then", registrationFinished)
}
