package handlerUserAgeGroups

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerUserAgeGroups."

const (
	qryGetAll = `SELECT * FROM et_user_age_groups`
	qryGet    = `SELECT * FROM et_user_age_groups WHERE id = $1`
	qryCreate = `INSERT INTO et_user_age_groups (age_group)
					VALUES ($1)
					RETURNING id`
	qryUpdate = `UPDATE et_user_age_groups 
					SET age_group = $1
					WHERE id = $3`
	qryDelete = `DELETE FROM et_user_age_groups WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.UserAgeGroups]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.UserAgeGroups]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.UserAgeGroups{} },
		NewRecord:   func() *models.UserAgeGroups { return &models.UserAgeGroups{} },
		IDDest:      func(record *models.UserAgeGroups) any { return &record.ID },
		SetID:       func(record *models.UserAgeGroups, id int) { record.ID = id },
		CreateArgs:  func(record *models.UserAgeGroups) []any { return []any{record.AgeGroup} },
		UpdateArgs:  func(id int, record *models.UserAgeGroups) []any { return []any{record.AgeGroup, id} },
	})
	return h
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
}

// RegisterRoutesPublic registers only the read-only routes (GET) for public access
func (h *Handler) RegisterRoutesPublic(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL, h.GetAll).Methods("GET")
	r.HandleFunc(baseURL+"/{id:[0-9]+}", h.Get).Methods("GET")
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
