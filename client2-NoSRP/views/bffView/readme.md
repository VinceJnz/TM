# Backend For Frontend (BFF) Security Framework

<https://docs.duendesoftware.com/bff/>


## How to make this work for the user

1. Use a token to allow the user to auto login. Allow this token to have qa limited life-time which will mean the user will need to periodically re-establish the auto logon.
2. If the auto login is lost the user logs in with username and password + emailed token to reestablish the auto login token.

## How to make this work on the api and wasm client app

### Initial account creation or account reset

1. Present the user with a screen to enter their info (username, password, email address, and any other detail needed) and submit it to the api server.
    Set up a temporary session token to track the whole registration process and allow the process to timeout.
2. Send the user a verification token via email to verify the email address.
3. Allow the user to enter the verificaion token on the same registration screen by displaying a token input and submit it to the api server.
4. Store the user details on the api server and create a login token to send to the browser to use for auto login.
    This information should be encrypted, esp the password (Ideally the password should not be stored).

### Normal use (Auto login)

1. The user connects to the api server with their registered browser. no other authentication is needed.


