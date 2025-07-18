package handlerUserAccountStatus

import (
	"encoding/json"
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/templates/handlerStandardTemplate"
	"api-server/v2/models"
)

//const debugTag = "handlerUserAccountStatus."

// Have hard coded this handler as it is used for the processing of accounts and it needs to be always the same, i.e. not DB driven.
// The API is provided for the benefit of the UI.

type AccountStatus int

const (
	AccountNew AccountStatus = iota
	AccountVerified
	AccountActive
	AccountDisabled
	AccountResetRequired
	AccountForDeletion
)

type Handler struct {
	appConf *appCore.Config
	list    []models.UserAccountStatus
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf,
		list: []models.UserAccountStatus{
			{ID: int(AccountNew), Status: "Account New", Description: "A new account that has just been created by a user. It is not yet verified or activated. Needs to be activated by an admin."},
			{ID: int(AccountVerified), Status: "Account Verified", Description: "The email address has been verified. An Admin now needs to activate the account."},
			{ID: int(AccountActive), Status: "Account Active", Description: "An account that has been activated, and is currently active."},
			{ID: int(AccountDisabled), Status: "Account Disabled", Description: "An account that has been disabled."},
			{ID: int(AccountResetRequired), Status: "Account Reset Required", Description: "The account is flagged for a password reset. The user will be informed at the next login."},
			{ID: int(AccountForDeletion), Status: "Account For Deletion", Description: "The account is flagged for deletion. An admin will need to perform the deletion."},
		},
	}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.list)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := handlerStandardTemplate.GetID(w, r) - 1 // The slice starts at index 0
	if id < 0 || id >= len(h.list) {
		http.Error(w, "Index out of range", http.StatusBadRequest)
		return
	}
	dest := h.list[id]
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dest)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
