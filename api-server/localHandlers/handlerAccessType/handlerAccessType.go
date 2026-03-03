package handlerAccessType

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerAccessType."

const (
	qryGetAll = `SELECT * FROM et_access_type`
	qryGet    = `SELECT * FROM et_access_type WHERE id = $1`
	qryCreate = `INSERT INTO et_access_type (name, description)
					VALUES ($1, $2)
					RETURNING id`
	qryUpdate = `UPDATE et_access_type 
					SET name = $1, description = $2
					WHERE id = $3`
	qryDelete = `DELETE FROM et_access_type WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.AccessType]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.AccessType]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.AccessType{} },
		NewRecord:   func() *models.AccessType { return &models.AccessType{} },
		IDDest:      func(record *models.AccessType) any { return &record.ID },
		SetID:       func(record *models.AccessType, id int) { record.ID = id },
		CreateArgs:  func(record *models.AccessType) []any { return []any{record.Name, record.Description} },
		UpdateArgs:  func(id int, record *models.AccessType) []any { return []any{record.Name, record.Description, id} },
	})
	return h
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
}

// GetAll: retrieves and returns all records
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	h.crud.GetAll(w, r)
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.crud.Get(w, r)
}

// Create: adds a new record and returns the new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.crud.Create(w, r)
}

// Update: modifies the existing record identified by id and returns the updated record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	h.crud.Update(w, r)
}

// Delete: removes a record identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	h.crud.Delete(w, r)
}
