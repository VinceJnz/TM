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
	"strings"
	"time"

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
	client.Timeout = 15 * time.Second
	return client, nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, providedAuthCode string) (*oauth2.Token, error) {
	if providedAuthCode == "" {
		return nil, errors.New("Gmail authorization required; configure a token or GMAIL_AUTH_CODE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tok, err := config.Exchange(ctx, providedAuthCode)
	if err != nil {
		return nil, errors.New("Google rejected the Gmail authorization code")
	}
	return tok, nil
}

func RenewToken(config *oauth2.Config, tok *oauth2.Token, cacheFile string) *oauth2.Token {
	log.Printf(debugTag + "RenewToken Attempting to refresh token\n")

	// Use the oauth2 library's built-in token refresh
	tokenSource := config.TokenSource(context.Background(), tok)
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Print(debugTag + "RenewToken refresh failed")
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

	config, err := OAuthConfig(credential)
	if err != nil {
		return nil, err
	}

	client, err := getClient(tokenFile, config, authCode)
	if err != nil {
		return nil, err
	}
	return newGateway(config, client, from, debugEmail)
}

func OAuthConfig(credentials []byte) (*oauth2.Config, error) {
	config, err := google.ConfigFromJSON(credentials, gmail.GmailSendScope)
	if err != nil || config.ClientID == "" || config.ClientSecret == "" {
		return nil, errors.New("invalid Google OAuth client credentials")
	}
	config.Endpoint = google.Endpoint
	return config, nil
}

func NewWithToken(config *oauth2.Config, token *oauth2.Token, from, debugEmail string) (*Gateway, error) {
	client := config.Client(context.Background(), token)
	client.Timeout = 15 * time.Second
	return newGateway(config, client, from, debugEmail)
}

func newGateway(config *oauth2.Config, client *http.Client, from, debugEmail string) (*Gateway, error) {
	gmailService, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
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
	return s.SendMailContext(context.Background(), to, title, message)
}

func (s *Gateway) SendMailContext(ctx context.Context, to string, title string, message string) (bool, error) {
	// Create the message
	if s.debugEmailAddress != "" {
		log.Print(debugTag + "SendMail recipient redirected by test policy")
		to = s.debugEmailAddress
	}
	if strings.ContainsAny(s.from, "\r\n") || strings.ContainsAny(to, "\r\n") || strings.ContainsAny(title, "\r\n") {
		return false, errors.New("mail headers must be single lines")
	}
	msgStr := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.from, to, title, message)

	msg := []byte(msgStr)
	// Get raw
	gMessage := &gmail.Message{Raw: base64.URLEncoding.EncodeToString(msg)}

	// Send the message
	_, err := s.srv.Users.Messages.Send("me", gMessage).Context(ctx).Do()
	if err != nil {
		log.Print(debugTag + "SendMail failed")
		// SECURITY: Message body not logged as it may contain sensitive data (OTPs, tokens, PII)
		return false, err
	}
	return true, nil
}
