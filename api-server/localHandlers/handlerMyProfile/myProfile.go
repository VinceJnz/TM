package handlerMyProfile

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const debugTag = "handlerMyProfile."

type ProfileResponse struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Address       string    `json:"user_address"`
	MemberCode    string    `json:"member_code"`
	BirthDate     time.Time `json:"user_birth_date"`
	AccountHidden bool      `json:"user_account_hidden"`
}

type ProfileUpdateRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Address       string `json:"user_address"`
	BirthDate     string `json:"user_birth_date"`
	AccountHidden bool   `json:"user_account_hidden"`
}

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL, h.Get).Methods("GET")
	r.HandleFunc(baseURL, h.Update).Methods("PUT")
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}

	user, err := dbAuthTemplate.UserReadQry(debugTag+"Get ", h.appConf.Db, session.UserID)
	if err != nil {
		log.Printf("%sGet failed to load user %d: %v", debugTag, session.UserID, err)
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}

	resp := ProfileResponse{
		ID:            user.ID,
		Name:          user.Name,
		Username:      user.Username,
		Email:         user.Email.String,
		Address:       user.Address.String,
		MemberCode:    user.MemberCode.String,
		BirthDate:     user.BirthDate.Time,
		AccountHidden: user.AccountHidden.Bool,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("%sGet failed to encode response: %v", debugTag, err)
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}

	var req ProfileUpdateRequest
	if err := helpers.DecodeJSONBody(r, &req); err != nil {
		log.Printf("%sUpdate decode error: %v", debugTag, err)
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	birthDate, err := time.Parse(time.RFC3339, req.BirthDate)
	if err != nil {
		birthDate, err = time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			http.Error(w, "invalid birth date format", http.StatusBadRequest)
			return
		}
	}

	const qry = `UPDATE st_users
		SET name = $2,
			email = $3,
			user_address = $4,
			user_birth_date = $5,
			user_account_hidden = $6
		WHERE id = $1`

	if _, err := h.appConf.Db.Exec(
		qry,
		session.UserID,
		name,
		strings.TrimSpace(req.Email),
		strings.TrimSpace(req.Address),
		birthDate,
		req.AccountHidden,
	); err != nil {
		log.Printf("%sUpdate failed for user %d: %v", debugTag, session.UserID, err)
		http.Error(w, "failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("%sUpdate failed to encode response: %v", debugTag, err)
	}
}
