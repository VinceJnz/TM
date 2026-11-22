package handlerAccessScope

import (
	"api-server/v2/app/appCore"
	catalogHTTP "api-server/v2/localHandlers/catalog"
	"api-server/v2/localHandlers/helpers"
	catalogStore "api-server/v2/modelMethods/catalog"
	"api-server/v2/models"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerAccessScope."

type Handler struct {
	appConf *appCore.Config
	crud    *catalogHTTP.Handler[models.AccessScope]
}

func New(appConf *appCore.Config) *Handler {
	h := &Handler{appConf: appConf}
	h.crud = catalogHTTP.New(appConf, catalogStore.NewAccessScope(appConf.Db), false)
	return h
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h, "securityAccessScope")
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
