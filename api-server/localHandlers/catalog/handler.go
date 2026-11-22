package catalog

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/catalog"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/catalog"
	"errors"
	"net/http"
	"strings"
)

type Handler[T any] struct {
	App        *appCore.Config
	Store      store.Store[T]
	PublicRead bool
}

func New[T any](app *appCore.Config, s store.Store[T], publicRead bool) *Handler[T] {
	return &Handler[T]{app, s, publicRead}
}
func (h *Handler[T]) authorize(w http.ResponseWriter, r *http.Request) bool {
	if h.PublicRead && r.Method == "GET" {
		return true
	}
	session := helpers.RequiredSession(w, r, h.App.SessionIDKey)
	if session == nil {
		return false
	}
	if !strings.EqualFold(session.AccessScope, "any") {
		helpers.WriteError(w, 403, "forbidden", "Catalog administration requires any scope")
		return false
	}
	return true
}
func (h *Handler[T]) GetAll(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	p, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	rows, info, err := h.Store.List(r.Context(), p)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteResourcePage(w, rows, info)
}
func (h *Handler[T]) Get(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	v, err := h.Store.Get(r.Context(), id)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, []T{v})
}
func (h *Handler[T]) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *Handler[T]) Update(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }
func (h *Handler[T]) save(w http.ResponseWriter, r *http.Request, update bool) {
	if !h.authorize(w, r) {
		return
	}
	id := 0
	if update {
		id = helpers.RouteID(w, r)
		if id == 0 {
			return
		}
	}
	var v T
	if err := helpers.DecodeJSONBody(r, &v); err != nil {
		helpers.WriteError(w, 400, "invalid_request", "Invalid catalog request")
		return
	}
	result, err := (service.Service[T]{Store: h.Store}).Save(r.Context(), id, v)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			helpers.WriteError(w, 422, "invalid_catalog", "Check the name and field lengths")
		} else {
			helpers.WriteDataError(w, err)
		}
		return
	}
	if !update {
		helpers.WriteJSON(w, 201, store.ID(result))
	} else {
		helpers.WriteJSON(w, 200, result)
	}
}
func (h *Handler[T]) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	id := helpers.RouteID(w, r)
	if id == 0 {
		return
	}
	if err := h.Store.Delete(r.Context(), id); err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, nil)
}
