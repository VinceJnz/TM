# token renew instructions

1. Copy the entire URL from the logs
2. Paste it into your web browser (Chrome, Edge, Firefox, etc.)
3. Sign in to Google with your account (e.g. something@gmail.com)
4. Click "Allow" when Google asks for permissions.
Note: The page itself won't load (you'll see "This site can't be reached" or similar), but the URL in the address bar should contain the authorization code you need.

5. Look at the URL in your browser's address bar after you authorize
    * The browser will redirect to something like `http://localhost/?state=state-token&code=...`
    * Copy the part after `code=` (e.g. somthing like `4/0............................Q_A`)

6. Past the code into the console.
You should see something like:
`2026/01/08 22:23:XX gmail.saveToken()2 ... Saving credential file to: /etc/certs/gmail-tokens/client_token.json`

