# WebAuthn registration process

Here’s a clear summary of the WebAuthn registration process and the data exchanged between your API server (the Relying Party) and a WASM (WebAssembly) client (typically running in the browser):

## Process Steps

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
* The server generates a challenge and other WebAuthn options, and stores the sessionData (in DB or memory) associated with the user's session/cookie.
* The server responds with the PublicKeyCredentialCreationOptions.

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
* The WASM client receives the options and calls the browser’s WebAuthn API:

```go
navigator.credentials.create({ publicKey: ... })
```
* The browser prompts the user to use a security key, fingerprint, etc.
* The browser returns a credential attestation (containing the public key and proof of possession).

4. Client Sends Attestation to Server (Client → Server)
* The WASM client sends the attestation response to the API server to finish registration.

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
* The API server retrieves the previously stored sessionData using the session/cookie.
* The server calls FinishRegistration, verifies the attestation, and if valid:
    * Creates the user in the database
    * Stores the credential (public key, credential ID, etc.) with the user

Response:
``` json
{ "status": "ok" }
```

## Summary Table

| Step	| Who	            | Action	                 | Data Exchanged                                       |
|---    |---                |---                         |---                                                   |
| 1	    | Client → Server	| Begin registration	     | User info (username, email, etc.)                    |
| 2	    | Server → Client	| Send registration options	 | PublicKeyCredentialCreationOptions (challenge, etc.) |
| 3	    | Client (browser)	| Call WebAuthn API	         | -                                                    |
| 4	    | Client → Server	| Finish registration	     | Attestation response (credential)                    |
| 5	    | Server	        | Verify & store	         | User and credential in DB                            |

Key Points:

* The server only stores the user in the DB after successful attestation verification in FinishRegistration.
* sessionData must be stored by the server between steps 2 and 5, referenced by a session/cookie.
* The WASM client acts as a bridge between your UI and the browser’s WebAuthn API.
