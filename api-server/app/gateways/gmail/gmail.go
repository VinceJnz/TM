package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const debugTag = "gmail."

type Gateway struct {
	srv  *gmail.Service
	from string
}

//https://developers.google.com/gmail/api/quickstart/go
//https://pkg.go.dev/google.golang.org/api@v0.63.0/gmail/v1
//https://console.cloud.google.com/

//gmail.Handler.SendMail()1 ... Could not send mail> googleapi: Error 403: Request had insufficient authentication scopes.
//https://stackoverflow.com/questions/65946707/googleapi-error-403-request-had-insufficient-authentication-scopes-more-detai

//Set the correct google api permissions
//https://developers.google.com/gmail/api/reference/rest/v1/users.messages/send

// Retrieve a token, saves the token, then returns the generated client.
func getClient(tokenFile string, config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	//
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		log.Printf(debugTag+"getClient()1 ... Token file not found %v", err)
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}

	//It is possible something else is needed to to auto renew tokens. (See sub-folder demo)
	//if tok.Expiry.Before(time.Now()) {
	//	log.Printf(debugTag + "getClient()2 need to renew new access token\n")
	//According to <https://github.com/googleapis/google-api-go-client/issues/111> the following can be used for refreshing the token
	//config.TokenSource(context.TODO(), tok)
	//if tok.Expiry.Before(time.Now()) {
	//tok = RenewToken(config, tok, cacheFile)
	//tok = RenewToken(config, tok, tokenFile)
	//}
	// The config.Client should renew tokens when they expire.
	//return config.Client(context.Background(), tok)

	// The config.Client will automatically renew tokens when they expire
	// as long as the refresh token is valid
	client := config.Client(context.Background(), tok)
	return client
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// The following generates a URL that the user can follow to create or renew a token
	// According the <https://github.com/googleapis/google-api-go-client/issues/111> using "oauth2.ApprovalForce" is needed to get the token to auto renew???
	//authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	log.Printf(debugTag+"Handler.getTokenFromWeb()1 ... Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	//The following waits for the user to paste a token into stdin
	//The token is obtained in the previous step
	var authCode string

	// Check if auth code is provided via environment variable
	authCode = os.Getenv("GMAIL_AUTH_CODE")

	if authCode == "" {
		// Fall back to stdin
		if _, err := fmt.Scan(&authCode); err != nil {
			log.Fatalf(debugTag+"Handler.getTokenFromWeb()2 ... Unable to read authorization code %v", err)
		}
	}

	tok, err := config.Exchange(context.TODO(), authCode, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	if err != nil {
		log.Fatalf(debugTag+"Handler.getTokenFromWeb()3 ... Unable to retrieve token from web %v", err)
	}
	return tok
}

// RenewToken renew the token ???????
func RenewToken1(config *oauth2.Config, tok *oauth2.Token, cacheFile string) *oauth2.Token {

	urlValue := url.Values{"client_id": {config.ClientID}, "client_secret": {config.ClientSecret}, "refresh_token": {tok.RefreshToken}, "grant_type": {"refresh_token"}}
	log.Printf(debugTag+"RenewToken()1 urlValue = %v\n", urlValue)

	resp, err := http.PostForm("https://www.googleapis.com/oauth2/v3/token", urlValue)
	if err != nil {
		//log.Panic("Error when renew token %v", err)
		log.Printf(debugTag+"RenewToken()2 Error when renewing token %v\n", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf(debugTag+"RenewToken()3 body = %s\n", body)
	//var refresh_token RefreshToken
	var refresh_token oauth2.Token
	json.Unmarshal([]byte(body), &refresh_token)

	log.Printf(debugTag+"RenewToken()4 refresh_token = %+v\n", refresh_token)

	then := time.Now()
	//then = then.Add(time.Duration(refresh_token.ExpiresIn) * time.Second)
	then = then.Add(24 * 6 * time.Hour)

	tok.Expiry = then
	tok.AccessToken = refresh_token.AccessToken
	saveToken(cacheFile, tok)

	return tok
}

func RenewToken(config *oauth2.Config, tok *oauth2.Token, cacheFile string) *oauth2.Token {
	log.Printf(debugTag + "RenewToken()1 Attempting to refresh token\n")

	// Use the oauth2 library's built-in token refresh
	tokenSource := config.TokenSource(context.Background(), tok)
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Printf(debugTag+"RenewToken()2 Error refreshing token: %v\n", err)
		log.Printf(debugTag + "RenewToken()3 You may need to re-authorize. Delete the token file and restart.\n")
		return tok // Return original token, let the caller handle the failure
	}

	log.Printf(debugTag + "RenewToken()4 Token refreshed successfully\n")
	saveToken(cacheFile, newToken)
	return newToken
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Printf(debugTag+"tokenFromFile()1 ... Token file not found %v", err)
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	log.Printf(debugTag+"saveToken()2 ... Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf(debugTag+"Handler.saveToken()2 ... Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func New(credentialsFile, tokenFile, from string) *Gateway {
	ctx := context.Background()
	credential, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatalf(debugTag+"New()1 ... Unable to read client secret json file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	//config, err := google.ConfigFromJSON(credential, drive.DriveMetadataReadonlyScope) //This is the config from the demo and only provides read permissions
	//config, err := google.ConfigFromJSON(credential, `https://www.googleapis.com/auth/gmail.send`)
	config, err := google.ConfigFromJSON(credential, gmail.GmailSendScope) //This is a non restricted scope
	if err != nil {
		log.Fatalf(debugTag+"New()2 ... Unable to parse client secret json file to config: %v", err)
	}

	client := getClient(tokenFile, config)

	//gmailService, err := gmail.NewService(ctx, option.WithAPIKey("AIza..."))
	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		log.Fatalln(debugTag+"New()3 ... Unable to retrieve Gmail client:", err)
	}

	log.Println(debugTag + "New()4 ... Gmail client created")
	return &Gateway{
		srv:  gmailService,
		from: from,
	}
}

func (s *Gateway) SendMail(to string, title string, message string) (bool, error) {
	// Create the message
	msgStr := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.from, to, title, message)
	msg := []byte(msgStr)
	// Get raw
	gMessage := &gmail.Message{Raw: base64.URLEncoding.EncodeToString(msg)}

	// Send the message
	//_, err := srv.Users.Messages.Send("me", gMessage).Do()
	_, err := s.srv.Users.Messages.Send("me", gMessage).Do()
	if err != nil {
		log.Println(debugTag+"Handler.SendMail()1 ... Could not send mail>", err)
		return false, err
	}
	return true, nil
}

func (s *Gateway) SendMail2(from string, to string, title string, message string) (bool, error) {
	// Create the message
	msgStr := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, title, message)
	msg := []byte(msgStr)
	// Get raw
	gMessage := &gmail.Message{Raw: base64.URLEncoding.EncodeToString(msg)}

	// Send the message
	//_, err := srv.Users.Messages.Send("me", gMessage).Do()
	_, err := s.srv.Users.Messages.Send("me", gMessage).Do()
	if err != nil {
		log.Println(debugTag+"Handler.SendMail()1 ... Could not send mail>", err)
		return false, err
	}
	return true, nil
}

func (s *Gateway) Demo(emailAddress string) (bool, error) {
	// Send a demo message
	if emailAddress == "" {
		return false, errors.New(debugTag + "Handler.Demo()1 ... error: emailAddress is empty")
	}
	from := emailAddress
	to := emailAddress
	ret, err := s.SendMail2(from, to, "Gmail With Go", "It worked!")
	return ret, err
}
