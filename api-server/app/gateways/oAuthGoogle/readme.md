# OAuth

<https://auth0.com/intro-to-iam/what-is-oauth-2>
<https://pkg.go.dev/golang.org/x/oauth2>
<https://oauth.net/code/go/>




## Major code components

✅ WASM Client

* Login button → redirect to API server.
* Handle callback (if token is returned directly).
* Store token (or rely on cookie).
* Attach token/cookie to API requests.

✅ API Server

* Configure oauth2.Config.
* /oauth/login → redirect to provider.
* /oauth/callback → exchange code for token, fetch user info.
* Create session (JWT or cookie).
* Middleware to protect API routes.


What is the advantages/disadvantages of the session/token handling options? 
* Option 1: HTTP-only cookies - Works well for browser base clients.
* Option 2: JWT

