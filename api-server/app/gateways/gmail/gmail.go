package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const debugTag = "gmail."

type Gateway struct {
	srv               *gmail.Service
	from              string
	debugEmailAddress string
	oauthConfig       *oauth2.Config
}

//https://developers.google.com/gmail/api/quickstart/go
//https://pkg.go.dev/google.golang.org/api@v0.63.0/gmail/v1
//https://console.cloud.google.com/

//gmail.Handler.SendMail ... Could not send mail> googleapi: Error 403: Request had insufficient authentication scopes.
//https://stackoverflow.com/questions/65946707/googleapi-error-403-request-had-insufficient-authentication-scopes-more-detai

//Set the correct google api permissions
//https://developers.google.com/gmail/api/reference/rest/v1/users.messages/send

// Retrieve a token, saves the token, then returns the generated client.
func getClient(tokenFile string, config *oauth2.Config, authCode string) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		log.Printf(debugTag+"getClient ... Token file not found %v", err)
		tok, err = getTokenFromWeb(config, authCode)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenFile, tok); err != nil {
			return nil, err
		}
	}

	// The config.Client will automatically renew tokens when they expire
	// as long as the refresh token is valid
	client := config.Client(context.Background(), tok)
	return client, nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, providedAuthCode string) (*oauth2.Token, error) {
	// The following generates a URL that the user can follow to create or renew a token
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	log.Printf(debugTag+"Handler.getTokenFromWeb ... Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	//The following waits for the user to paste a token into stdin
	//The token is obtained in the previous step
	var authCode string

	// Check if auth code was provided by caller
	authCode = providedAuthCode

	if authCode == "" {
		// Fall back to stdin
		if _, err := fmt.Scan(&authCode); err != nil {
			return nil, fmt.Errorf("unable to read authorization code: %w", err)
		}
	}

	tok, err := config.Exchange(context.Background(), authCode, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

func RenewToken(config *oauth2.Config, tok *oauth2.Token, cacheFile string) *oauth2.Token {
	log.Printf(debugTag + "RenewToken Attempting to refresh token\n")

	// Use the oauth2 library's built-in token refresh
	tokenSource := config.TokenSource(context.Background(), tok)
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Printf(debugTag+"RenewToken Error refreshing token: %v\n", err)
		log.Printf(debugTag + "RenewToken You may need to re-authorize. Delete the token file and restart.\n")
		return tok // Return original token, let the caller handle the failure
	}

	log.Printf(debugTag + "RenewToken Token refreshed successfully\n")
	if err := saveToken(cacheFile, newToken); err != nil {
		log.Printf(debugTag+"RenewToken Error saving refreshed token: %v\n", err)
	}
	return newToken
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Printf(debugTag+"tokenFromFile ... Token file not found %v", err)
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	log.Printf(debugTag+"saveToken ... Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("unable to encode oauth token: %w", err)
	}
	return nil
}

func New(credentialsFile, tokenFile, from, debugEmail, authCode string) (*Gateway, error) {
	ctx := context.Background()
	if credentialsFile == "" {
		return nil, errors.New("gmail credentials file is required")
	}
	if tokenFile == "" {
		return nil, errors.New("gmail token file is required")
	}
	credential, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret json file: %w", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credential, gmail.GmailSendScope) //This is a non restricted scope
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret json file to config: %w", err)
	}

	client, err := getClient(tokenFile, config, authCode)
	if err != nil {
		return nil, err
	}
	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %w", err)
	}

	log.Println(debugTag + "New ... Gmail client created")
	return &Gateway{
		srv:               gmailService,
		from:              from,
		debugEmailAddress: debugEmail,
		oauthConfig:       config,
	}, nil
}

// RenewURL returns the Google OAuth2 authorisation URL the admin must visit to
// obtain a fresh auth code when the token needs to be bootstrapped or renewed.
// The returned URL is intended to be displayed in the admin UI only.
func (s *Gateway) RenewURL() string {
	if s.oauthConfig == nil {
		return ""
	}
	return s.oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// SendMail sends an email using the Gmail API. It constructs the email message, encodes it, and sends it through the Gmail service.
// It returns true if the email was sent successfully, or false along with an error if there was an issue.
func (s *Gateway) SendMail(to string, title string, message string) (bool, error) {
	// Create the message
	if s.debugEmailAddress != "" {
		log.Printf(debugTag+"Handler.SendMail ... Debug email address configured, overriding recipient. Original to: %s, title: %s", to, title)
		to = s.debugEmailAddress
	}
	msgStr := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.from, to, title, message)

	msg := []byte(msgStr)
	// Get raw
	gMessage := &gmail.Message{Raw: base64.URLEncoding.EncodeToString(msg)}

	// Send the message
	_, err := s.srv.Users.Messages.Send("me", gMessage).Do()
	if err != nil {
		log.Println(debugTag+"Handler.SendMail ... Could not send mail>", err, "to:", to, "subject:", title)
		// SECURITY: Message body not logged as it may contain sensitive data (OTPs, tokens, PII)
		return false, err
	}
	return true, nil
}
