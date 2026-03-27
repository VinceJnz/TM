package handlerGmailCert

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/gateways/gmail"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

const debugTag = "handlerGmailCert."
const maxUploadBytes = 64 * 1024 // 64 KB max for cert JSON files

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL, h.Get).Methods("GET")
	r.HandleFunc(baseURL, h.Update).Methods("PUT")
	r.HandleFunc(baseURL+"/renew-url", h.GetRenewURL).Methods("GET")
}

// CertStatus is returned by GET and describes the current state of both cert files.
type CertStatus struct {
	EmailAddr      string    `json:"email_addr"`
	TokenFile      string    `json:"token_file"`
	TokenExists    bool      `json:"token_exists"`
	TokenModified  time.Time `json:"token_modified,omitempty"`
	SecretFile     string    `json:"secret_file"`
	SecretExists   bool      `json:"secret_exists"`
	SecretModified time.Time `json:"secret_modified,omitempty"`
}

// UpdateRequest is the PUT request body.
type UpdateRequest struct {
	Type    string `json:"type"`    // "token" or "secret"
	Content string `json:"content"` // raw JSON content to write
}

// UpdateResponse is returned on a successful PUT.
type UpdateResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func fileInfo(path string) (exists bool, modified time.Time) {
	info, err := os.Stat(path)
	if err != nil {
		return false, time.Time{}
	}
	return true, info.ModTime()
}

// RenewURLResponse is returned by GET /gmailcert/renew-url.
type RenewURLResponse struct {
	URL     string `json:"url"`
	Message string `json:"message"`
}

// GetRenewURL returns the Google OAuth2 authorisation URL the admin should visit
// to obtain a fresh auth code for token bootstrapping or renewal.
func (h *Handler) GetRenewURL(w http.ResponseWriter, r *http.Request) {
	if h.appConf.EmailSvc == nil {
		http.Error(w, "email service not initialised", http.StatusServiceUnavailable)
		return
	}
	authURL := h.appConf.EmailSvc.RenewURL()
	if authURL == "" {
		http.Error(w, "unable to generate renewal URL — secret file may be missing", http.StatusInternalServerError)
		return
	}
	log.Printf("%sGetRenewURL renewal URL generated for admin", debugTag)
	writeJSON(w, http.StatusOK, RenewURLResponse{
		URL:     authURL,
		Message: "Visit this URL in a browser, authorise access, then paste the returned auth code into the token file update form above.",
	})
}

// Get returns the current status of the Gmail cert files (existence, modification time).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tokenExists, tokenMod := fileInfo(h.appConf.Settings.EmailToken)
	secretExists, secretMod := fileInfo(h.appConf.Settings.EmailSecret)

	status := CertStatus{
		EmailAddr:      h.appConf.Settings.EmailAddr,
		TokenFile:      h.appConf.Settings.EmailToken,
		TokenExists:    tokenExists,
		TokenModified:  tokenMod,
		SecretFile:     h.appConf.Settings.EmailSecret,
		SecretExists:   secretExists,
		SecretModified: secretMod,
	}

	writeJSON(w, http.StatusOK, status)
}

// Update accepts a new token or secret file content, writes it atomically, then reinitialises
// the email service so the change takes effect immediately without a restart.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var targetPath string
	switch req.Type {
	case "token":
		targetPath = h.appConf.Settings.EmailToken
	case "secret":
		targetPath = h.appConf.Settings.EmailSecret
	default:
		http.Error(w, `type must be "token" or "secret"`, http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content must not be empty", http.StatusBadRequest)
		return
	}

	// Validate that the uploaded content is well-formed JSON before touching the filesystem.
	if !json.Valid([]byte(req.Content)) {
		http.Error(w, "content is not valid JSON", http.StatusBadRequest)
		return
	}

	// Write atomically: create a temp file in the same directory, then rename.
	dir := filepath.Dir(targetPath)
	tmpFile, err := os.CreateTemp(dir, ".gmailcert-upload-*")
	if err != nil {
		log.Printf("%sUpdate failed to create temp file in %s: %v", debugTag, dir, err)
		http.Error(w, "failed to write cert file", http.StatusInternalServerError)
		return
	}
	tmpPath := tmpFile.Name()

	// Ensure the temp file is cleaned up if the rename does not happen.
	cleanupTmp := true
	defer func() {
		tmpFile.Close()
		if cleanupTmp {
			os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(0600); err != nil {
		// Non-fatal on some platforms; log and continue.
		log.Printf("%sUpdate could not chmod temp file: %v", debugTag, err)
	}

	if _, err := io.WriteString(tmpFile, req.Content); err != nil {
		log.Printf("%sUpdate failed to write to temp file: %v", debugTag, err)
		http.Error(w, "failed to write cert file", http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, targetPath); err != nil {
		log.Printf("%sUpdate failed to rename %s -> %s: %v", debugTag, tmpPath, targetPath, err)
		http.Error(w, "failed to install cert file", http.StatusInternalServerError)
		return
	}
	cleanupTmp = false // rename succeeded; nothing left to clean up

	log.Printf("%sUpdate %s cert file written to %s", debugTag, req.Type, targetPath)

	// Re-initialise the email service with the updated file(s) so the change takes effect
	// immediately. NOTE: EmailSvc replacement is not mutex-protected; the brief race window
	// risks at most one in-flight email using the old client, which is acceptable here.
	debugEmail := ""
	if h.appConf.Settings.DevMode {
		debugEmail = h.appConf.Settings.EmailDebugAddr
	}
	newSvc, err := gmail.New(
		h.appConf.Settings.EmailSecret,
		h.appConf.Settings.EmailToken,
		h.appConf.Settings.EmailAddr,
		debugEmail,
		"",
	)
	if err != nil {
		log.Printf("%sUpdate cert written but email service reinit failed: %v", debugTag, err)
		writeJSON(w, http.StatusOK, UpdateResponse{
			Message: fmt.Sprintf("%s cert file updated. Warning: email service reinit failed — a server restart may be required.", req.Type),
			Type:    req.Type,
		})
		return
	}

	h.appConf.EmailSvc = newSvc
	log.Printf("%sUpdate email service reinitialized after %s cert update", debugTag, req.Type)

	writeJSON(w, http.StatusOK, UpdateResponse{
		Message: fmt.Sprintf("%s cert file updated and email service reinitialized successfully.", req.Type),
		Type:    req.Type,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("%sfailed to encode JSON response: %v", debugTag, err)
	}
}
