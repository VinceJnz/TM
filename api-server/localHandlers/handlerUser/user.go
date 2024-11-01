package handlerUser

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"api-server/v2/app"
	"api-server/v2/models"

	"github.com/gorilla/mux"
)

const debugTag = "handlerUser."

type Handler struct {
	appConf *app.Config
}

func New(appConf *app.Config) *Handler {
	return &Handler{appConf: appConf}
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	records := []models.User{}
	err := h.appConf.Db.Select(&records, `SELECT id, name, username, email FROM st_users`)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf(debugTag+"GetAll()2 %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Get()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	record := models.User{}
	err = h.appConf.Db.Get(&record, "SELECT id, name, username, email FROM st_users WHERE id = $1", id)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var record models.User
	json.NewDecoder(r.Body).Decode(&record)

	err := h.appConf.Db.QueryRow(
		"INSERT INTO st_users (name, username, email) VALUES ($1, $2, $3) RETURNING id",
		record.Name, record.Username, record.Email).Scan(&record.ID)
	if err != nil {
		log.Printf("%v.Create()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Update()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	var record models.User
	json.NewDecoder(r.Body).Decode(&record)
	record.ID = id

	_, err = h.appConf.Db.Exec("UPDATE st_users SET name = $1, username = $2, email = $3 WHERE id = $4",
		record.Name, record.Username, record.Email, record.ID)
	if err != nil {
		log.Printf("%v.Update()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%v.Delete()1 %v\n", debugTag, err)
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	_, err = h.appConf.Db.Exec("DELETE FROM st_users WHERE id = $1", id)
	if err != nil {
		log.Printf("%v.Delete()2 %v\n", debugTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
