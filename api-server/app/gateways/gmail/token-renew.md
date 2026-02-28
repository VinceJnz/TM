# token renew instructions

1. delete or rename the file <client_token.json>
2. Restart the api server (IMPORTANT: in interactive mode: docker compose --profile interactive up --build -d)
3. Go to the api-server console and start the api-server process
4. In the start up logs  - look for the URL line like the example below - Copy the entire URL from the logs
```log
apiserver  | 2026/01/27 10:37:05 gmail.Handler.getTokenFromWeb()1 ... Go to the following link in your browser then type the authorization code:
apiserver  | https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=687205587230958857023948752tjtp8398fwo4faw98faw8faw48faw48hf.apps.googleusercontent.com&prompt=consent&redirect_uri=http%3A%2F%2Flocalhost&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.send&state=state-token
```

5. Paste it into your web browser (Chrome, Edge, Firefox, etc.)
6. Sign in to Google with your account (e.g. something@gmail.com)
7. Click "Allow" when Google asks for permissions.
Note: The page itself won't load (you'll see "This site can't be reached" or similar), but the URL in the address bar should contain the authorization code you need.

8. Look at the URL in your browser's address bar after you authorize
    * The browser will redirect to something like `http://localhost/?state=state-token&code=...`
    * Copy the part after `code=` (e.g. somthing like `4/0............................Q_A`) and before `&...`

9. Past the code into the console.
You should see something like:
`2026/01/08 22:23:XX gmail.saveToken()2 ... Saving credential file to: /etc/certs/gmail-tokens/client_token.json`

