package handlerAccessScope

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerAccessScope."

const (
	qryGetAll = `SELECT * FROM et_access_scope`
	qryGet    = `SELECT * FROM et_access_scope WHERE id = $1`
	qryCreate = `INSERT INTO et_access_scope (name, description)
					VALUES ($1, $2)
					RETURNING id`
	qryUpdate = `UPDATE et_access_scope 
					SET name = $1, description = $2
					WHERE id = $3`
	qryDelete = `DELETE FROM et_access_scope WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.AccessScope]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.AccessScope]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.AccessScope{} },
		NewRecord:   func() *models.AccessScope { return &models.AccessScope{} },
		IDDest:      func(record *models.AccessScope) any { return &record.ID },
		SetID:       func(record *models.AccessScope, id int) { record.ID = id },
		CreateArgs:  func(record *models.AccessScope) []any { return []any{record.Name, record.Description} },
		UpdateArgs:  func(id int, record *models.AccessScope) []any { return []any{record.Name, record.Description, id} },
	})
	return h
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	h.crud.GetAll(w, r)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.crud.Get(w, r)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.crud.Create(w, r)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	h.crud.Update(w, r)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	h.crud.Delete(w, r)
}
