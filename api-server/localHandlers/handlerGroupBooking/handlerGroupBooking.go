package handlerGroupBooking

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerGroupBooking."

const (
	qryGetAll = `SELECT id, group_name, owner_id, created, modified FROM at_group_bookings`
	qryGet    = `SELECT id, group_name, owner_id, created, modified FROM at_group_bookings WHERE id = $1`
	qryCreate = `INSERT INTO at_group_bookings (group_name, owner_id) VALUES ($1, $2) RETURNING id`
	qryUpdate = `UPDATE at_group_bookings SET group_name = $1, owner_id = $2 WHERE id = $3`
	qryDelete = `DELETE FROM at_group_bookings WHERE id = $1`
)

type Handler struct {
	appConf *appCore.Config
	crud    *dbStandardTemplate.ResourceCRUD[models.GroupBooking]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = dbStandardTemplate.NewResourceCRUD(dbStandardTemplate.ResourceCRUDConfig[models.GroupBooking]{
		DebugTag: debugTag,
		Db:       h.appConf.Db,
		Queries: dbStandardTemplate.CRUDQueries{
			GetAll: qryGetAll,
			Get:    qryGet,
			Create: qryCreate,
			Update: qryUpdate,
			Delete: qryDelete,
		},
		NewListDest: func() any { return &[]models.GroupBooking{} },
		NewRecord:   func() *models.GroupBooking { return &models.GroupBooking{} },
		IDDest:      func(record *models.GroupBooking) any { return &record.ID },
		SetID:       func(record *models.GroupBooking, id int) { record.ID = id },
		CreateArgs:  func(record *models.GroupBooking) []any { return []any{record.GroupName, record.OwnerID} },
		UpdateArgs:  func(id int, record *models.GroupBooking) []any { return []any{record.GroupName, record.OwnerID, id} },
	})
	return h
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
}

// GetAll: retrieves and returns all group bookings
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	h.crud.GetAll(w, r)
}

// Get: retrieves and returns a single group booking identified by id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.crud.Get(w, r)
}

// Create: adds a new group booking
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.crud.Create(w, r)
}

// Update: modifies an existing group booking
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	h.crud.Update(w, r)
}

// Delete: removes a group booking identified by id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	h.crud.Delete(w, r)
}
