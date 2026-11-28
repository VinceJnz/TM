package handlerSecurityGroup

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/security"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/security"
	"api-server/v2/models"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

type Handler struct {
	app   *appCore.Config
	store store.Store
}

func New(app *appCore.Config) *Handler { return &Handler{app, store.Store{DB: app.Db}} }
func (h *Handler) RegisterRoutes(r *mux.Router, path string) {
	helpers.AddRouteGroup(r, path, h, "securityGroup")
}
func (h *Handler) actor(w http.ResponseWriter, r *http.Request) int {
	s := helpers.RequiredSession(w, r, h.app.SessionIDKey)
	if s == nil {
		return 0
	}
	if !strings.EqualFold(s.AccessScope, "any") {
		helpers.WriteError(w, 403, "forbidden", "Security administration requires any scope")
		return 0
	}
	return s.UserID
}
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	if h.actor(w, r) == 0 {
		return
	}
	p, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	rows, info, err := h.store.Groups(r.Context(), p)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	response := make([]models.GroupDTO, len(rows))
	for i, row := range rows {
		response[i] = models.GroupDTO(row)
	}
	helpers.WriteResourcePage(w, response, info)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if h.actor(w, r) == 0 {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	v, err := h.store.Group(r.Context(), id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, []models.GroupDTO{models.GroupDTO(v)})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }
func (h *Handler) save(w http.ResponseWriter, r *http.Request, update bool) {
	actor := h.actor(w, r)
	if actor == 0 {
		return
	}
	id := 0
	if update {
		id = helpers.RouteID(w, r)
		if id == 0 {
			return
		}
	}
	var input models.GroupWrite
	if err := helpers.DecodeJSONBody(r, &input); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid security payload")
		return
	}
	result, err := (service.Service{Store: h.store}).SaveGroup(r.Context(), actor, id, input)
	if err != nil {
		h.failure(w, err)
		return
	}
	if !update {
		helpers.WriteJSON(w, 201, result.ID)
		return
	}
	helpers.WriteJSON(w, 200, models.GroupDTO(result))
}
func (h *Handler) failure(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrBuiltInGroup) {
		helpers.WriteError(w, 409, "built_in_group", "Built-in security groups cannot be deleted")
	} else if errors.Is(err, store.ErrReferencedGroup) {
		helpers.WriteError(w, 409, "group_in_use", "Remove or reassign group memberships and grants before deleting the group")
	} else if errors.Is(err, service.ErrValidation) {
		helpers.WriteError(w, 422, "invalid_security_change", "A group name is required; name and description must be at most 45 characters")
	} else {
		helpers.WriteDataError(w, err)
	}
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor := h.actor(w, r)
	if actor == 0 {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	if err := (service.Service{Store: h.store}).DeleteGroup(r.Context(), actor, id); err != nil {
		h.failure(w, err)
		return
	}
	helpers.WriteJSON(w, 200, nil)
}
