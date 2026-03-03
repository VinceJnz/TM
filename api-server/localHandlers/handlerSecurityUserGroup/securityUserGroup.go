package handlerSecurityUserGroup

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
	qryGetAll = `SELECT stug.id, stug.user_id, stu.name as user_name, stug.group_id, stg.name as group_name
					FROM st_user_group stug
						JOIN st_users stu ON stu.id=stug.user_id
						JOIN st_group stg ON stg.id=stug.group_id`
	qryGet    = `SELECT id, user_id, group_id FROM st_user_group WHERE id = $1`
	qryCreate = `INSERT INTO st_user_group (user_id, group_id) VALUES ($1, $2) RETURNING id`
	qryUpdate = `UPDATE st_user_group SET user_id = $1, group_id = $2 WHERE id = $3`
	qryDelete = `DELETE FROM st_user_group WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.UserGroup]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.UserGroup]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.UserGroup{} },
		NewRecord:   func() *models.UserGroup { return &models.UserGroup{} },
		IDDest:      func(record *models.UserGroup) any { return &record.ID },
		SetID:       func(record *models.UserGroup, id int) { record.ID = id },
		CreateArgs:  func(record *models.UserGroup) []any { return []any{record.UserID, record.GroupID} },
		UpdateArgs:  func(id int, record *models.UserGroup) []any { return []any{record.UserID, record.GroupID, id} },
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
