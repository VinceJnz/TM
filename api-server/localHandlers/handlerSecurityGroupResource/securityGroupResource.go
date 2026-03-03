package handlerSecurityGroupResource

import (
	"net/http"

	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"

	"github.com/gorilla/mux"
)

const debugTag = "handlerSecurityGroupResource."

const (
	qryGetAll = `SELECT stgr.id, stgr.group_id, stg.name AS group_name, resource_id, etr.name AS resource, access_level_id, etal.name AS access_level, access_type_id, etat.name AS access_type, stgr.role 
					FROM st_group_resource stgr
						JOIN st_group stg ON stg.id=stgr.group_id
						JOIN et_resource etr ON etr.id=stgr.resource_id
						JOIN et_access_level etal ON etal.id=stgr.access_level_id
						JOIN et_access_type etat ON etat.id=stgr.access_type_id`
	qryGet    = `SELECT id, group_id, resource_id, access_level_id, access_type_id, role FROM st_group_resource WHERE id = $1`
	qryCreate = `INSERT INTO st_group_resource (group_id, resource_id, access_level_id, access_type_id, role) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	qryUpdate = `UPDATE st_group_resource SET group_id = $1, resource_id = $2, access_level_id = $3, access_type_id = $4, role = $5 WHERE id = $6`
	qryDelete = `DELETE FROM st_group_resource WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.GroupResource]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.GroupResource]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.GroupResource{} },
		NewRecord:   func() *models.GroupResource { return &models.GroupResource{} },
		IDDest:      func(record *models.GroupResource) any { return &record.ID },
		SetID:       func(record *models.GroupResource, id int) { record.ID = id },
		CreateArgs: func(record *models.GroupResource) []any {
			return []any{record.GroupID, record.ResourceID, record.AccessLevelID, record.AccessTypeID, record.Role}
		},
		UpdateArgs: func(id int, record *models.GroupResource) []any {
			return []any{record.GroupID, record.ResourceID, record.AccessLevelID, record.AccessTypeID, record.Role, id}
		},
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
