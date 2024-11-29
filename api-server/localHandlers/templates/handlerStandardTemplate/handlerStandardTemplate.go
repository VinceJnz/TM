package handlerStandardTemplate

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

const debugTag = "handlerStandardTemplate."

// GetAll: retrieves and returns all records
func GetAll(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest interface{}, query string, args ...interface{}) {
	err := Db.Select(dest, query)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf(debugStr+"GetAll()2 %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dest)
}

// Get: retrieves and returns a single record identified by id
func Get(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest interface{}, query string, args ...interface{}) {
	err := Db.Get(dest, query, args)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("%v.Get()2 %v\n", debugStr, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dest)
}

// Create: adds a new record and returns the new record
func Create(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest interface{}, query string, args ...interface{}) {
	//log.Printf(debugTag+debugStr+"Create()1 dest=%+v", dest)
	//err := json.NewDecoder(r.Body).Decode(dest)
	//if err != nil {
	//	log.Printf(debugTag+debugStr+"Create()2 err=%+v", err)
	//	http.Error(w, "Invalid request payload", http.StatusBadRequest)
	//	return
	//}
	log.Printf(debugTag+debugStr+"Create()3 dest=%+v", dest)

	err := Db.QueryRow(query, args...).Scan(dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dest)
}

// Update: modifies the existing record identified by id and returns the updated record
func Update(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest interface{}, query string, args ...interface{}) {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		log.Printf(debugStr+"Update()1 dest=%+v", dest)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	result, err := Db.Exec(query, args...)
	if err != nil {
		log.Printf(debugStr+"Update()2 result=%+v", result)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dest)
}

// Delete: removes a record identified by id
func Delete(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest interface{}, query string, args ...interface{}) {
	result, err := Db.Exec(query, args...)
	if err != nil {
		log.Printf(debugTag+debugStr+"Delete()1 result=%+v", result)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func GetID(w http.ResponseWriter, r *http.Request) int {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return 0
	}
	return id
}
