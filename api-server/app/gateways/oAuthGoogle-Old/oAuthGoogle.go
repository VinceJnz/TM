package oAuthGoogle

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	store       = sessions.NewCookieStore([]byte("super-secret-key"))
	oauthConfig *oauth2.Config
)

func init() {
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}
}

func randString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")

	state := randString(32)
	session.Values["oauth-state"] = state
	session.Save(r, w)

	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")

	// Validate state
	state := r.URL.Query().Get("state")
	if state != session.Values["oauth-state"] {
		http.Error(w, "Invalid state", http.StatusForbidden)
		return
	}
	delete(session.Values, "oauth-state")

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code returned", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch user info
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(w, "Failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store user in session
	session.Values["user_id"] = userInfo["sub"] // unique ID
	session.Values["email"] = userInfo["email"]
	session.Values["name"] = userInfo["name"]
	session.Save(r, w)

	http.Redirect(w, r, "http://localhost:8081", http.StatusFound)
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")
	uid, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp := map[string]interface{}{
		"user_id": uid,
		"email":   session.Values["email"],
		"name":    session.Values["name"],
	}
	json.NewEncoder(w).Encode(resp)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")
	session.Options.MaxAge = -1 // delete
	session.Save(r, w)
	http.Redirect(w, r, "http://localhost:8081", http.StatusFound)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/callback", callbackHandler)
	r.HandleFunc("/logout", logoutHandler)
	r.HandleFunc("/me", meHandler)

	log.Println("API listening on :8080")
	http.ListenAndServe(":8080", r)
}
