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

```
userHandle and User ID do not match
```

from the Go WebAuthn package (`github.com/go-webauthn/webauthn/webauthn`) occurs **during the assertion (authentication) process** when the `userHandle` returned from the client does **not match** the user ID known to the server for the credential being used.

---

### ✅ What this means

In the WebAuthn protocol:

* During authentication (`FinishLogin()`), the **authenticator** sends back a `userHandle`.
* The **server** uses the credential ID (from the request) to **look up the stored credential**, including the user ID it was registered with.
* The server then **compares**:

  ```
  response.UserHandle == storedCredential.User.ID()
  ```

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
