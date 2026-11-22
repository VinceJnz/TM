package handlerUserAgeGroups

import (
	"api-server/v2/app/appCore"
	catalogHTTP "api-server/v2/localHandlers/catalog"
	"api-server/v2/localHandlers/helpers"
	catalogStore "api-server/v2/modelMethods/catalog"
	"api-server/v2/models"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerUserAgeGroups."

type Handler struct {
	appConf *appCore.Config
	crud    *catalogHTTP.Handler[models.UserAgeGroups]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = catalogHTTP.New(appConf, catalogStore.NewUserAgeGroups(appConf.Db), true)
	return h
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h, "userAgeGroups")
}

// RegisterRoutesPublic registers only the read-only routes (GET) for public access
func (h *Handler) RegisterRoutesPublic(r *mux.Router, baseURL string) {
	helpers.CapabilityRoute(r, baseURL, "userAgeGroups", h.GetAll).Methods("GET")
	helpers.CapabilityRoute(r, baseURL+"/{id:[0-9]+}", "userAgeGroups", h.Get).Methods("GET")
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
