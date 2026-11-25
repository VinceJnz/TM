## Sysadmin gateway credentials

Open **System Administrator → Gateway Credentials**. The screen uses the existing
`gmailcert` capability with `any` scope. GET grants access to status and PUT
permits changes.

- **Refresh Gmail token** exchanges the saved refresh token and installs the
  result immediately. A failed exchange leaves the saved file and loaded client
  in place.
- **Reconnect Gmail** starts a Google consent flow for a missing or revoked
  refresh token. Configure a Google OAuth web client and register the full API
  callback URL as an authorized redirect URI. Set `GMAIL_RENEW_REDIRECT_URL`
  to that exact URL, for example
  `https://trips.example.com/api/gmailcert/callback` for an `/api` prefix.
  The callback must receive the same signed-in Sysadmin session. Start and
  complete consent against the same API instance within ten minutes.
- Configure `EMAIL_SECRET` and `EMAIL_TOKEN` as file paths. Replacement
  requires writable parent directories on persistent storage. When replacing
  the OAuth client JSON, refresh with its matching token or reconnect to
  activate the new client. Complete that step before restarting the API.
- Stripe and Google sign-in show startup configuration status. Rotate those
  credentials in the provider console, update the deployment secrets, and
  restart the API.

# Gmail API Info

## Testing password reset in development

In the test API environment file, set `DEV_MODE=true` and
`EMAIL_DEV_MODE_ADDR` to an inbox you can access.
Recreate the API container and enter the registered email of any active test
account with a password in **Forgot password?**. The code goes to the test
inbox and applies to the account whose email you entered.

## API Info

<https://developers.google.com/gmail/api/quickstart/go>
<https://pkg.go.dev/google.golang.org/api@v0.63.0/gmail/v1>
<https://console.cloud.google.com/>

GMAIL API Error
`gmail.Handler.SendMail ... Could not send mail> googleapi: Error 403: Request had insufficient authentication scopes.`
<https://stackoverflow.com/questions/65946707/googleapi-error-403-request-had-insufficient-authentication-scopes-more-detai>

Seting for google gmail api SEND permissions
<https://developers.google.com/gmail/api/reference/rest/v1/users.messages/send>

## api packages

### Google golang oAuth2

<https://developers.google.com/identity/protocols/oauth2>

<https://pkg.go.dev/golang.org/x/oauth2/google>

<https://github.com/googleapis/google-api-go-client/issues/111>

<https://stackoverflow.com/questions/52825464/how-to-use-google-refresh-token-when-the-access-token-is-expired-in-go>

<https://stackoverflow.com/questions/65756962/gmail-api-token-expired-how-to-get-new-one>

<https://community.n8n.io/t/google-calendar-oauth2-api-token-expiring-every-once-in-a-while/11336/5>

<https://www.labnol.org/google-oauth-refresh-token-220423>

<https://auth0.com/blog/refresh-tokens-what-are-they-and-when-to-use-them/>

Possible solution to token renewal?????
<https://pkg.go.dev/golang.org/x/oauth2#ReuseTokenSource>

### golang oAuth2

<https://pkg.go.dev/golang.org/x/oauth2@v0.0.0-20220524215830-622c5d57e401#Config>

## Alert - change in operation of API (4-May-2022)
Hello Google OAuth Developer,

We are writing to inform you that OAuth out-of-band (OOB) flow will be deprecated on October 3, 2022, to protect users from phishing and app impersonation attacks.

Please check our recent blog post about Making Google OAuth interactions safer for more information.

What do I need to do?
No action is required on your part as apps using OOB in testing mode will not be affected by this change. However, we strongly recommend you to migrate to safer methods as these apps will be immediately blocked when switching to in production status.

If you want to publish your app(s) to production, follow these instructions:

Determine your app(s) client type from your Google Cloud project by following the client links below.
Migrate your app(s) to a more secure alternative method by following the instructions in the blog post linked above for your client type.
The following OAuth client(s) are using the OOB flow in test mode.

OAuth client list:

Project ID: tripmanager-347208
Client: 897941135228-lfuqaj79rqndv2c44se4ev3qohs4rsjd.apps.googleusercontent.com
Thanks for choosing Google OAuth.

— The Google OAuth Developer Team


## How to auth access to gmail server

After authenticating in the browser, copy the code value from the URL:
`http://localhost/?state=state-token&code=4/0xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxYw&scope=https://www.googleapis.com/auth/gmail.send`


Paste only the value after code= (up to the & or end of string) into your Docker terminal when prompted.
`4/0xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxYw`
