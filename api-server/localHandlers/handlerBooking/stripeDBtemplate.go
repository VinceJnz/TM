package handlerBooking

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

// Get: retrieves and returns a single record identified by id
func Get(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest any, query string, args ...any) error {
	err := Db.Get(dest, query, args...)
	if err == sql.ErrNoRows {
		newErr := fmt.Errorf(debugTag+debugStr+" database error: %w", err)
		http.Error(w, "Record not found", http.StatusNotFound)
		return newErr
	} else if err != nil {
		log.Printf(debugTag+debugStr+"Get()2 err=%v\n", err)
		newErr := fmt.Errorf(debugTag+debugStr+" database error: %w", err)
		http.Error(w, newErr.Error(), http.StatusInternalServerError)
		return newErr
	}
	return nil
}

// Update: modifies the existing record identified by id and returns the updated record
func Update(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest any, query string, args ...any) error {
	//if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
	//	log.Printf(debugTag+debugStr+"Update()1 err=%+v, dest=%+v", err, dest)
	//	http.Error(w, "Invalid request payload", http.StatusBadRequest)
	//	return
	//}

	tx, err := Db.Beginx() // Start transaction
	if err != nil {
		http.Error(w, debugTag+debugStr+"Create()1: Could not start transaction", http.StatusInternalServerError)
		return err
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		tx.Rollback() // Rollback on error
		log.Printf(debugTag+debugStr+"Update()2 err=%+v, result=%+v", err, result)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	err = tx.Commit() // Commit on success
	if err != nil {
		http.Error(w, debugTag+debugStr+"Create()3: Could not commit transaction", http.StatusInternalServerError)
		return err
	}

	//w.WriteHeader(http.StatusOK)
	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(dest) // Not sure why this is returning a copy of the record (dest) that was sent in.
	return nil
}
