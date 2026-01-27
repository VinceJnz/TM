# Gmail API Info

## API Info

<https://developers.google.com/gmail/api/quickstart/go>
<https://pkg.go.dev/google.golang.org/api@v0.63.0/gmail/v1>
<https://console.cloud.google.com/>

GMAIL API Error
`gmail.Handler.SendMail()1 ... Could not send mail> googleapi: Error 403: Request had insufficient authentication scopes.`
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
