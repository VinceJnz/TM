# Backend For Frontend (BFF) Security Framework

<https://docs.duendesoftware.com/bff/>
<https://auth0.com/blog/the-backend-for-frontend-pattern-bff/>

## Go packages
<https://www.talentica.com/blogs/backend-for-frontend-bff-authentication-what-it-is-and-how-to-implement-it-in-go/>
<https://www.talentica.com/blogs/backend-for-frontend-bff-authentication-in-go-part-2/>
<https://dev.to/mehulgohil/backend-for-frontend-authentication-in-go-2e98>
<https://github.com/adityaeka26/go-bff>



## Cookie settings
 Set the session cookie to strict and http only and secure

### CSRF attack protection
Effective CSRF attack protection relies on these pillars:

* Using Same-Site=strict Cookies
* Requiring a specific header to be sent on every API request (IE: x-csrf=1)
* having a cors policy that restricts the cookies only to a list of white-listed origins.


## Interface and steps

### User/device registration

1. Registration begin request
    1. Receive request with user data
    2. Check for existing user/device and begin user reregister device
    3. If new user then begin register the user
    4. Send email for token to verify identity
    5. Store user data etc in temporary pool using token as key

2. Registration finish request
    1. Receive request with token
    2. Check token
    3. Finish registration and notify user
    4. Send application cookie to browser (this will have a long term expirery)

### User/device login

1. This will use the application cookie to provide the client access to the application api/data via the wasm client
    When the cookie expires the user will need to re-register.
