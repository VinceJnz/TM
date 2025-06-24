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
* BeginRegistration: The server generates a challenge and other WebAuthn options, and stores the sessionData (in DB or memory e.g. a pool) associated with the user's temporary session/cookie.
  * sessionData: <https://pkg.go.dev/github.com/go-webauthn/webauthn@v0.13.0/webauthn#SessionData>
* The server responds with the PublicKeyCredentialCreationOptions and a temporary session cookie/token.
  * options: <https://pkg.go.dev/github.com/go-webauthn/webauthn@v0.13.0/protocol#CredentialCreation> - This gets sent to the client.
    * PublicKeyCredentialCreationOptions: https://pkg.go.dev/github.com/go-webauthn/webauthn@v0.13.0/protocol#PublicKeyCredentialCreationOptions
    * CredentialMediationRequirement: https://pkg.go.dev/github.com/go-webauthn/webauthn@v0.13.0/protocol#CredentialMediationRequirement

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



## WebAuthn information

### Infomation to be stored

When WebAuthn registration is successful, the server should store the following information in the database for each credential:

**User Table**
* User information (if not already stored):
  * User ID (internal, e.g., integer primary key)
  * Username, email, etc.
  * WebAuthn user handle (a unique, opaque identifier, e.g., UUID or random string, used as WebAuthnID)

**WebAuthn Credentials Table** (one row per credential)
* User ID (foreign key to user table)
* Credential ID (unique, base64-encoded string)
* Public Key (base64-encoded or PEM string)
* AAGUID (Authenticator Attestation GUID, base64-encoded string)
* Sign Count (integer, from the authenticator)
* Credential Type (usually "public-key")
* Attestation Format (optional, for auditing)
* Created At (timestamp)

**Example Table Schema:**
```sql
CREATE TABLE webauthn_credentials (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    aaguid TEXT,
    sign_count INTEGER NOT NULL,
    credential_type TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Summary:**

Store the user’s WebAuthn handle in the user table, and for each credential, store: user ID, credential ID, public key, AAGUID, sign count, credential type, and created timestamp in a separate credentials table.
This ensures you can authenticate users, support multiple credentials per user, and manage credentials securely.

## Information that should not be stored
When storing WebAuthn registration data in your database, you should NOT store:

* Private keys: Never store the authenticator’s private key. Only the public key is needed for verification.
* Raw attestation objects (unless you have a specific auditing or compliance need): These can be large and are not required for authentication.
* Sensitive client data: Do not store unnecessary client-side information such as user agent, IP address, or geolocation unless required for security/audit purposes.
* Temporary session data: Do not persist SessionData used during registration/login ceremonies; it should only be kept in memory or a temporary store and deleted after the ceremony.
* Passwords or PINs: WebAuthn is passwordless; do not store user PINs or authenticator secrets.
* Unencrypted sensitive fields: If you must store anything sensitive (rare for WebAuthn), ensure it is encrypted at rest.

**Summary:**

Only store what is needed for authentication: user handle, credential ID, public key, AAGUID, sign count, credential type, and user association.
Never store private keys, temporary session data, or unnecessary sensitive client information.