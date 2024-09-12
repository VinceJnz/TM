package handler2

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
)

// BaseRecord is an interface that defines methods common to all records
type BaseRecord interface {
	GetID() int
	SetID(id int)
}

// BaseHandler provides common functionality for CRUD operations
type BaseHandler[T any] struct {
	db     *sqlx.DB
	table  string
	fields string
	scan   func(rows *sql.Rows) (T, error)
	get    func(row *sql.Row) (T, error)
	insert func(record *T) (int, error)
	update func(record *T) error
}

// NewBaseHandler creates a new BaseHandler
func NewBaseHandler[T BaseRecord](
	db *sqlx.DB,
	table string,
	fields string,
	scanFunc func(rows *sql.Rows) (T, error),
	getFunc func(row *sql.Row) (T, error),
	insertFunc func(record *T) (int, error),
	updateFunc func(record *T) error,
) *BaseHandler[T] {
	return &BaseHandler[T]{
		db:     db,
		table:  table,
		fields: fields,
		scan:   scanFunc,
		get:    getFunc,
		insert: insertFunc,
		update: updateFunc,
	}
}

// GetAll retrieves all records from the table
func (h *BaseHandler[T]) GetAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT " + h.fields + " FROM " + h.table)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		record, err := h.scan(rows)
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
func (h *BaseHandler[T]) Get(w http.ResponseWriter, r *http.Request, id int) {
	record, err := h.get(h.db.QueryRow("SELECT "+h.fields+" FROM "+h.table+" WHERE id = $1", id))
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
func (h *BaseHandler[T]) Create(w http.ResponseWriter, r *http.Request) {
	var record T
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	id, err := h.insert(&record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	record.SetID(id)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// Update updates an existing record
func (h *BaseHandler[T]) Update(w http.ResponseWriter, r *http.Request, id int) {
	var record T
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	record.SetID(id)
	if err := h.update(&record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// Delete removes a record
func (h *BaseHandler[T]) Delete(w http.ResponseWriter, r *http.Request, id int) {
	_, err := h.db.Exec("DELETE FROM "+h.table+" WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
