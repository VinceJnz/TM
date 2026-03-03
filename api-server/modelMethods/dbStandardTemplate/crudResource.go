package dbStandardTemplate

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type CRUDQueries struct {
	GetAll string
	Get    string
	Create string
	Update string
	Delete string
}

type ResourceCRUDConfig[T any] struct {
	DebugTag string
	Db       *sqlx.DB
	Queries  CRUDQueries

	NewListDest func() any
	NewRecord   func() *T
	IDDest      func(*T) any
	SetID       func(*T, int)
	CreateArgs  func(*T) []any
	UpdateArgs  func(int, *T) []any
}

type ResourceCRUD[T any] struct {
	cfg ResourceCRUDConfig[T]
}

func NewResourceCRUD[T any](cfg ResourceCRUDConfig[T]) *ResourceCRUD[T] {
	return &ResourceCRUD[T]{cfg: cfg}
}

func (rc *ResourceCRUD[T]) GetAll(w http.ResponseWriter, r *http.Request) {
	GetAll(w, r, rc.cfg.DebugTag, rc.cfg.Db, rc.cfg.NewListDest(), rc.cfg.Queries.GetAll)
}

func (rc *ResourceCRUD[T]) Get(w http.ResponseWriter, r *http.Request) {
	id := GetID(w, r)
	Get(w, r, rc.cfg.DebugTag, rc.cfg.Db, rc.cfg.NewListDest(), rc.cfg.Queries.Get, id)
}

func (rc *ResourceCRUD[T]) Create(w http.ResponseWriter, r *http.Request) {
	record := rc.cfg.NewRecord()
	if err := json.NewDecoder(r.Body).Decode(record); err != nil {
		log.Printf(rc.cfg.DebugTag+"Create err=%+v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	Create(w, r, rc.cfg.DebugTag, rc.cfg.Db, rc.cfg.IDDest(record), rc.cfg.Queries.Create, rc.cfg.CreateArgs(record)...)
}

func (rc *ResourceCRUD[T]) Update(w http.ResponseWriter, r *http.Request) {
	record := rc.cfg.NewRecord()
	id := GetID(w, r)
	if err := json.NewDecoder(r.Body).Decode(record); err != nil {
		log.Printf(rc.cfg.DebugTag+"Update dest=%+v", *record)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	rc.cfg.SetID(record, id)
	Update(w, r, rc.cfg.DebugTag, rc.cfg.Db, record, rc.cfg.Queries.Update, rc.cfg.UpdateArgs(id, record)...)
}

func (rc *ResourceCRUD[T]) Delete(w http.ResponseWriter, r *http.Request) {
	id := GetID(w, r)
	Delete(w, r, rc.cfg.DebugTag, rc.cfg.Db, nil, rc.cfg.Queries.Delete, id)
}
