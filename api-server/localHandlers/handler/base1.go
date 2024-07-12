package handler

import (
	//"api-server/v2/localHandlers/handler"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
)

// BaseHandler provides common functionality for CRUD operations
type BaseHandler[T any] struct {
	DB     *sqlx.DB
	table  string
	fields string
	ID     int
}

// NewBaseHandler creates a new BaseHandler
func NewBaseHandler[T any](db *sqlx.DB, table string, fields string) *BaseHandler[T] {
	return &BaseHandler[T]{DB: db, table: table, fields: fields}
}

// GetAll retrieves all records from the table
func (h *BaseHandler[T]) GetAll(w http.ResponseWriter, r *http.Request, scanFunc func(*sql.Rows) (T, error)) {
	rows, err := h.DB.Query("SELECT " + h.fields + " FROM " + h.table)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		record, err := scanFunc(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

// Get retrieves a single record by ID
func (h *BaseHandler[T]) Get(w http.ResponseWriter, r *http.Request, id int, scanFunc func(*sql.Row) (T, error)) {
	var record T
	err := h.DB.QueryRow("SELECT "+h.fields+" FROM "+h.table+" WHERE id = $1", id).Scan(scanFunc)
	if err == sql.ErrNoRows {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// Create inserts a new record
func (h *BaseHandler[T]) Create(w http.ResponseWriter, r *http.Request, insertFunc func(*T) (int, error), record T) {
	if err := json.NewDecoder(r.Body).Decode(record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	id, err := insertFunc(&record)
	//_, err := insertFunc(record) //????
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	record.ID = id //????
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// Update updates an existing record
func (h *BaseHandler[T]) Update(w http.ResponseWriter, r *http.Request, id int, updateFunc func(*T) error, record *T) {
	if err := json.NewDecoder(r.Body).Decode(record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	//record.ID = id //????
	//BaseHandler(record).ID = id //????
	if err := updateFunc(record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// Delete removes a record
func (h *BaseHandler[T]) Delete(w http.ResponseWriter, r *http.Request, id int) {
	_, err := h.DB.Exec("DELETE FROM "+h.table+" WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
