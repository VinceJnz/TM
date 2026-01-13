package handlerBooking

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

// Get: retrieves and returns a single record identified by id
func Get(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest any, query string, args ...any) {
	err := Db.Get(dest, query, args...)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf(debugTag+debugStr+"Get()2 err=%v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//log.Printf(debugTag+debugStr+"Get()2 dest=%+v\n", dest)

	//w.WriteHeader(http.StatusOK)
	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(dest)
}

// Update: modifies the existing record identified by id and returns the updated record
func Update(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest any, query string, args ...any) {
	//if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
	//	log.Printf(debugTag+debugStr+"Update()1 err=%+v, dest=%+v", err, dest)
	//	http.Error(w, "Invalid request payload", http.StatusBadRequest)
	//	return
	//}

	tx, err := Db.Beginx() // Start transaction
	if err != nil {
		http.Error(w, debugTag+debugStr+"Create()1: Could not start transaction", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		tx.Rollback() // Rollback on error
		log.Printf(debugTag+debugStr+"Update()2 err=%+v, result=%+v", err, result)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.Commit() // Commit on success
	if err != nil {
		http.Error(w, debugTag+debugStr+"Create()3: Could not commit transaction", http.StatusInternalServerError)
		return
	}

	//w.WriteHeader(http.StatusOK)
	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(dest) // Not sure why this is returning a copy of the record (dest) that was sent in.
}
