# WebAuthn processes

## WebAuthn registration process

Here’s a summary of the WebAuthn registration process and the data exchanged between your API server (the Relying Party) and a WASM (WebAssembly) client (typically running in the browser):

### Process Steps

1. User Initiates Registration (Client → Server)
* The user fills out a registration form (e.g., username, email) and clicks "Register".
* The WASM client sends a registration request (usually a POST) to your API server, including the user info.

Example request:
```json
POST /webauthn/register/begin
{
  "username": "alice",
  "email": "alice@example.com"
}
```

2. Server Generates Registration Options (Server → Client)
* The API server receives the request, creates a new (not yet persisted) user object, and calls BeginRegistration.
* The server generates a challenge and other WebAuthn options, and stores the sessionData (in DB or memory e.g. a pool) associated with the user's temporary session/cookie.
* The server responds with the PublicKeyCredentialCreationOptions and a temporary session cookie/token.

Example response:
```json
{
  "publicKey": {
    "challenge": "...",
    "rp": { ... },
    "user": { ... },
    "pubKeyCredParams": [ ... ],
    ...
  }
}
```

3. Client Calls WebAuthn API (Client-side)
* The browserautomatically stores the temporary session cookie/token.
* The WASM client receives the options plus the temporary session cookie/token and calls the browser’s WebAuthn API:

For registration:
```go
navigator.credentials.create({ publicKey: ... })
```
* The browser prompts the user to use a security key, fingerprint, etc.
* The browser returns a credential attestation (containing the public key and proof of possession).

4. Client Sends Attestation to Server (Client → Server)
* The WASM client sends the attestation response to the API server to finish registration.
* The browser automatically sends temporary session cookie/token.

Example request:
```json
POST /webauthn/register/finish
{
  "id": "...",
  "rawId": "...",
  "response": { ... },
  "type": "public-key"
}
```

5. Server Verifies and Stores User (Server-side)
* The API server retrieves the previously stored sessionData using the temporary session/cookie.
* The server calls FinishRegistration, verifies the attestation, and if valid:
    * Creates the user in the database
    * Stores the credential (public key, credential ID, etc.) with the user

Response:
``` json
{ "status": "ok" }
```

### Summary Table

| Step	| Who	             | Action	                    | Data Exchanged                                                        |
|---    |---               |---                         |---                                                                    |
| 1	    | Client → Server	 | Begin registration	        | User info (username, email, etc.)                                     |
| 2	    | Server → Client	 | Send registration options	| PublicKeyCredentialCreationOptions (challenge, etc.) + temp session   |
| 3	    | Client (browser) | Call WebAuthn API	        | -                                                                     |
| 4	    | Client → Server	 | Finish registration	      | Attestation response (credential) with temp session                   |
| 5	    | Server	         | Verify & store	            | User and credential in DB                                             |

Key Points:

* The server only stores the user in the DB after successful attestation verification in FinishRegistration.
* sessionData must be stored by the server between steps 2 and 5, referenced by a temporary session/cookie.
* The WASM client acts as a bridge between your UI and the browser’s WebAuthn API.


## WebAuthn login process

Here’s a summary of the WebAuthn login (authentication) process and the data exchanged between your API server (Relying Party) and a WASM client/browser:

### Process Steps

1. User Initiates Login (Client → Server)
The user enters their username (or email) and clicks "Login".
The WASM client sends a login request (usually a POST) to your API server, including the username.

Example request:
```json
POST /webauthn/login/begin
{
  "username": "alice"
}
```


2. Server Generates Login Options (Server → Client)
The API server looks up the user and their registered credentials.
The server calls BeginLogin(user) to generate a challenge and options, and stores the sessionData (in DB, memory, or a pool) associated with a temporary session token or cookie.
The server responds with the PublicKeyCredentialRequestOptions and sets the temporary session token/cookie.

Example response:
```json
{
  "publicKey": {
    "challenge": "...",
    "allowCredentials": [ ... ],
    ...
  }
}
```
(Plus a temporary session cookie or token for tracking the ceremony)

3. Client Calls WebAuthn API (Client-side)
The WASM client receives the options and calls the browser’s WebAuthn API:
```go
navigator.credentials.get({ publicKey: ... })
```
The browser prompts the user to use their authenticator (security key, fingerprint, etc.).
The browser returns an assertion (proof of possession of the credential).

4. Client Sends Assertion to Server (Client → Server)
The WASM client sends the assertion response to the API server to finish login.

Example request:
```json
POST /webauthn/login/finish
{
  "id": "...",
  "rawId": "...",
  "response": { ... },
  "type": "public-key"
}
```
(The temporary session token/cookie is sent automatically or as a field)

5. Server Verifies and Authenticates User (Server-side)
The API server retrieves the previously stored sessionData using the temporary session token/cookie.
The server calls FinishLogin(user, sessionData, r) to verify the assertion.
If valid, the server authenticates the user and issues a session/authentication token (e.g., sets a session cookie or returns a JWT).

Example response:
```json
{ "status": "ok", "message": "User authenticated" }
```

### Summary Table

| Step | Who	            | Action           	    | Data Exchanged                                                |
|---   |---               |---                    |---                                                            |
| 1	   | Client → Server  | Begin login	          | Username (or user reference)                                  |
| 2	   | Server → Client  | Send login options    | PublicKeyCredentialRequestOptions + temp session token/cookie |
| 3	   | Client (browser) | Call WebAuthn API     | -                                                             |
| 4	   | Client → Server  | Finish login	        | Assertion response (+ temp session token/cookie)              |
| 5	   | Server           | Verify & authenticate	| Issues session/auth token (cookie/JWT)                        |

Key Points:

The server only authenticates the user after successful assertion verification in FinishLogin.
sessionData must be stored by the server between steps 2 and 5, referenced by a temporary session/cookie or token.
The WASM client acts as a bridge between your UI and the browser’s WebAuthn API.




## Demo

<https://webauthn.io/>

<https://github.com/duo-labs/webauthn>


```js
navigator.credentials.store({
    'type': 'password',
    'id': 'alice',
    'password': 'VeryRandomPassword123456'
})
```


```js
navigator.credentials
    .get({ 'password': true })
    .then(credential => {
        if (!credential) {
            throw new Error('No credentials returned!')
        }

        let credentials = {
            'username': credential.id,
            'password': credential.password
        }

        return fetch('https://example.com/loginEndpoint', {
            method: 'POST',
            body: JSON.stringify(credentials),
            credentials: 'include'
        })
    })
    .then((response) => {
        ...
    })
```



## Debug notes from chatGPT

### Question

In golang I am using the package "github.com/go-webauthn/webauthn/webauthn" to do authentication.

I get the following error when trying to authenticate: userHandle and User ID do not match

How to I determine where the error is coming from, e.g. API server or client?


### Answer

The error message:
`userHandle and User ID do not match`

from the Go WebAuthn package (`github.com/go-webauthn/webauthn/webauthn`) occurs **during the assertion (authentication) process** when the `userHandle` returned from the client does **not match** the user ID known to the server for the credential being used.

---

### ✅ What this means

In the WebAuthn protocol:

* During authentication (`FinishLogin()`), the **authenticator** sends back a `userHandle`.
* The **server** uses the credential ID (from the request) to **look up the stored credential**, including the user ID it was registered with.
* The server then **compares**:

  `response.UserHandle == storedCredential.User.ID()`

If these **don’t match exactly**, the server throws:

> `userHandle and User ID do not match`

---

### 🔍 Where the error is happening

**This error is thrown on the API server**, specifically inside the `webauthn.FinishLogin()` function. It is **not a client-side error**.

---

### 🧪 How to investigate

Here’s how to find the cause:

#### 1. **Check what you're returning for `User.ID()`**

* The `User` object must implement the `webauthn.User` interface.
* Make sure the `ID()` method returns a **consistent and correct byte slice**.

Example:

```go
func (u *User) ID() []byte {
    return []byte(u.IDString) // or however you store the ID
}
```

Ensure the byte slice returned here is **exactly the same** as what was used during registration for this credential.

#### 2. **Check stored credential’s User ID**

* When storing credentials during registration, make sure the `Credential` struct (with `Authenticator`) is linked to the correct user ID.
* Some bugs happen if you're storing the wrong user ID alongside the credential.

#### 3. **Log the values**

Add debug logs before calling `webauthn.FinishLogin()`:

```go
log.Printf("Start FinishLogin with user ID: %x", user.ID())
```

Then inspect the `response.UserHandle` manually:

```go
log.Printf("UserHandle from client: %x", parsedAssertionResponse.Response.UserHandle)
```

To dig deeper, you might wrap or fork the library to add detailed logging.

---

### ⚠️ Common Mistakes

* Returning a new `[]byte` each time from `User.ID()` that isn't equal due to memory or encoding issues.
* Storing the credential without correctly associating it with the user's ID at registration time.
* Base64 encoding/decoding issues between frontend and backend.
* If you're storing user IDs as strings in your database, make sure you're not returning a mismatched string-to-byte conversion.

---

### 🧭 Summary

* ✅ **Error location**: Server side, in `FinishLogin()`
* 🔍 **Check**: That `User.ID()` matches the `userHandle` that was returned during login
* 🛠 **Solution**: Ensure consistent user ID handling across registration and authentication

Would you like help inspecting a specific snippet of your code (e.g., your `User` struct or how you handle credential registration)?




## Client code explanation notes from deepseek

This Go code is part of a WebAssembly (Wasm) client that implements the WebAuthn registration flow. It interacts with JavaScript APIs from Go using the syscall/js package. Here's a step-by-step explanation:

1. Initial Promise Handling

```go
then := js.FuncOf(func(this js.Value, args []js.Value) any {
    resp := args[0]
    jsonPromise := resp.Call("json")
```

* A JavaScript then callback is created to handle the response (resp) from a fetch request.
* resp.Call("json") converts the response to a JSON promise.

2. JSON Response Handling

```go
then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
    options := args[0]
    publicKey := options.Get("publicKey")
```

* Another then callback (then2) is created to handle the resolved JSON promise.
* It extracts the publicKey options from the response, which contains WebAuthn registration parameters.

3. Data Conversion

```go
publicKey.Set("challenge", decodeBase64URLToUint8Array(publicKey.Get("challenge").String()))
user := publicKey.Get("user")
user.Set("id", decodeBase64URLToUint8Array(user.Get("id").String()))
publicKey.Set("user", user)
```

* The WebAuthn challenge and user.id (which are base64url-encoded strings) are converted to Uint8Array as required by the WebAuthn API.
* The modified values are set back in the publicKey object.

4. WebAuthn API Call

```go
credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
    "publicKey": publicKey,
})
```

* Calls the WebAuthn API (navigator.credentials.create()) with the prepared publicKey options.
* This prompts the user to register a new authenticator (e.g., security key or biometric device).

5. Credential Handling

```go
then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
    cred := args[0]
    credJSON := js.Global().Get("JSON").Call("stringify", cred)
```

* A third then callback (then3) handles the credential result.
* The credential is converted to a JSON string.

6. Sending Result to Server

```go
js.Global().Get("fetch").Invoke(ApiURL+"/register/finish/", map[string]any{
    "method": "POST",
    "body":   credJSON,
    "headers": map[string]any{
        "Content-Type": "application/json",
    },
})
```

* Sends the credential to the server's /register/finish/ endpoint via a POST request.
* The credential is sent as JSON in the request body.

7. Promise Chaining
```go
credPromise.Call("then", then3)
jsonPromise.Call("then", then2)
respPromise.Call("then", then)
```

* The promises are chained together:
  * respPromise → then (handles initial response)
  * jsonPromise → then2 (handles JSON conversion)
  * credPromise → then3 (handles WebAuthn credential)

**Key Points:**
* The code uses Go's js package to interoperate with JavaScript.
* It follows the WebAuthn registration flow:
  * Receives options from the server.
  * Converts data formats.
  * Calls the WebAuthn browser API.
  * Sends the result back to the server.
* Each async operation is handled via JavaScript promises with Go callbacks.

This is a typical pattern for WebAuthn in Wasm, where Go manages the flow but delegates to browser APIs for cryptographic operations and user interaction.


## Firefox webAuthn requirements

*****************************************************************************
For firefox to work with soft passkeys, Windows hello need to be enabled.
This is also required for Edge.
*****************************************************************************

```log
2025/10/03 12:24:29 webAuthnRegisterView.WebAuthnRegistration()11 Calling navigator.credentials.create() 
2025/10/03 12:24:30 webAuthnRegisterView.WebAuthnRegistration()13 Error: The operation failed for an unknown transient reason 
2025/10/03 12:24:30 webAuthnRegisterView.WebAuthnRegistration()19 Failed: UnknownError - The operation failed for an unknown transient reason
```

This error is common in WebAuthn implementations and can have several causes. Here are the most likely issues and solutions.

1. Base64 Encoding Issues (Most Common)
Make sure you're properly encoding the challenge and other binary data:
```go
func prepareOptions(options map[string]interface{}) map[string]interface{} {
    // Ensure challenge is properly base64url encoded without padding
    if challenge, ok := options["challenge"].([]byte); ok {
        options["challenge"] = base64.RawURLEncoding.EncodeToString(challenge)
    }
    
    // Convert user ID if present
    if user, ok := options["user"].(map[string]interface{}); ok {
        if id, ok := user["id"].([]byte); ok {
            user["id"] = base64.RawURLEncoding.EncodeToString(id)
        }
    }
    
    // Handle excludeCredentials if present
    if exclude, ok := options["excludeCredentials"].([]map[string]interface{}); ok {
        for i, cred := range exclude {
            if id, ok := cred["id"].([]byte); ok {
                cred["id"] = base64.RawURLEncoding.EncodeToString(id)
                exclude[i] = cred
            }
        }
    }
    
    return options
}
```

2. Proper PublicKey Credential Creation
Here's a complete WebAuthn registration function:
```go
func WebAuthnRegistration(options js.Value) {
    log.Printf("WebAuthnRegistration()1 Starting registration")
    
    // Convert options to proper format
    publicKey := convertToPublicKey(options)
    
    log.Printf("WebAuthnRegistration()2 Converted options: %s", 
        js.Global().Get("JSON").Call("stringify", publicKey, js.Null(), 2).String())
    
    // Get navigator.credentials
    navigator := js.Global().Get("navigator")
    credentials := navigator.Get("credentials")
    
    log.Printf("WebAuthnRegistration()3 Calling navigator.credentials.create()")
    
    // Create the credential
    promise := credentials.Call("create", map[string]interface{}{
        "publicKey": publicKey,
    })
    
    // Handle the promise
    promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        credential := args[0]
        log.Printf("WebAuthnRegistration()14 Success: %s", 
            js.Global().Get("JSON").Call("stringify", credential, js.Null(), 2).String())
        
        // Handle successful registration
        handleSuccessfulRegistration(credential)
        return nil
    }))
    
    promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        err := args[0]
        log.Printf("WebAuthnRegistration()15 Error: %s", err.String())
        log.Printf("WebAuthnRegistration()16 Error name: %s", err.Get("name").String())
        log.Printf("WebAuthnRegistration()17 Error message: %s", err.Get("message").String())
        
        // More detailed error logging
        if !err.Get("message").IsUndefined() {
            log.Printf("WebAuthnRegistration()18 Full error: %v", err)
        }
        
        handleRegistrationError(err)
        return nil
    }))
}

func convertToPublicKey(options js.Value) map[string]interface{} {
    publicKey := make(map[string]interface{})
    
    // Required fields
    publicKey["rp"] = map[string]interface{}{
        "name": options.Get("rp").Get("name").String(),
        "id":   options.Get("rp").Get("id").String(),
    }
    
    publicKey["user"] = map[string]interface{}{
        "id":          decodeBase64URL(options.Get("user").Get("id").String()),
        "name":        options.Get("user").Get("name").String(),
        "displayName": options.Get("user").Get("displayName").String(),
    }
    
    publicKey["challenge"] = decodeBase64URL(options.Get("challenge").String())
    
    // Optional fields with checks
    if !options.Get("pubKeyCredParams").IsUndefined() {
        publicKey["pubKeyCredParams"] = options.Get("pubKeyCredParams")
    }
    
    if !options.Get("timeout").IsUndefined() {
        publicKey["timeout"] = options.Get("timeout").Int()
    } else {
        publicKey["timeout"] = 60000 // Default timeout
    }
    
    if !options.Get("authenticatorSelection").IsUndefined() {
        publicKey["authenticatorSelection"] = options.Get("authenticatorSelection")
    }
    
    if !options.Get("excludeCredentials").IsUndefined() {
        excludeCreds := []map[string]interface{}{}
        length := options.Get("excludeCredentials").Length()
        for i := 0; i < length; i++ {
            cred := options.Get("excludeCredentials").Index(i)
            excludeCreds = append(excludeCreds, map[string]interface{}{
                "type":       cred.Get("type").String(),
                "id":         decodeBase64URL(cred.Get("id").String()),
                "transports": cred.Get("transports"),
            })
        }
        publicKey["excludeCredentials"] = excludeCreds
    }
    
    return publicKey
}

func decodeBase64URL(s string) []byte {
    // Add padding if needed
    if len(s)%4 != 0 {
        padding := 4 - len(s)%4
        s += strings.Repeat("=", padding)
    }
    
    // Replace URL-safe characters
    s = strings.ReplaceAll(s, "-", "+")
    s = strings.ReplaceAll(s, "_", "/")
    
    decoded, err := base64.StdEncoding.DecodeString(s)
    if err != nil {
        log.Printf("Base64 decode error: %v", err)
        return []byte{}
    }
    return decoded
}
```


3. Check Firefox-Specific Issues
Firefox can be particular about:
```go
// Ensure you're using the correct origin
func checkEnvironment() {
    location := js.Global().Get("location")
    origin := location.Get("origin").String()
    protocol := location.Get("protocol").String()
    
    log.Printf("Current origin: %s", origin)
    log.Printf("Current protocol: %s", protocol)
    
    // WebAuthn requires HTTPS (except localhost)
    if protocol != "https:" && !strings.Contains(origin, "localhost") {
        log.Printf("WARNING: WebAuthn may not work without HTTPS")
    }
}
```


4. Add Better Error Handling

```go
func handleRegistrationError(err js.Value) {
    name := err.Get("name").String()
    message := err.Get("message").String()
    
    log.Printf("Registration failed: %s - %s", name, message)
    
    switch name {
    case "NotAllowedError":
        log.Printf("User cancelled the operation or no authenticator available")
    case "InvalidStateError":
        log.Printf("Authenticator was already registered")
    case "NotSupportedError":
        log.Printf("WebAuthn not supported by browser/authenticator")
    case "AbortError":
        log.Printf("Operation was aborted")
    case "UnknownError":
        log.Printf("Unknown error - check console for details")
        // Log additional details if available
        if !err.Get("details").IsUndefined() {
            log.Printf("Error details: %v", err.Get("details"))
        }
    default:
        log.Printf("Unexpected error type: %s", name)
    }
}
```


5. Debug Steps
Add these debug checks:
```go
func debugWebAuthnSupport() {
    navigator := js.Global().Get("navigator")
    
    // Check if WebAuthn is supported
    if navigator.Get("credentials").IsUndefined() {
        log.Printf("ERROR: WebAuthn not supported in this browser")
        return
    }
    
    // Check if create is available
    credentials := navigator.Get("credentials")
    if credentials.Get("create").IsUndefined() {
        log.Printf("ERROR: credentials.create not available")
        return
    }
    
    log.Printf("WebAuthn appears to be supported")
}
```

6. Common Firefox Fixes
Try adding these to your publicKey options:
```go
publicKey["attestation"] = "none"
publicKey["authenticatorSelection"] = map[string]interface{}{
    "authenticatorAttachment": "cross-platform",
    "requireResidentKey":      false,
    "userVerification":        "preferred",
}
```