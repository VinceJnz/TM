package handlerTripDifficulty

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/tripdifficulty"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const debugTag = "handlerTripDifficulty."

type store interface {
	List(context.Context) ([]models.TripDifficulty, error)
	Get(context.Context, int) (models.TripDifficulty, error)
	Create(context.Context, models.TripDifficulty) (models.TripDifficulty, error)
	Update(context.Context, int, models.TripDifficulty) (models.TripDifficulty, error)
	Delete(context.Context, int) error
}

type Handler struct {
	store store
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{store: tripdifficulty.New(appConf.Db)}
}

// RegisterRoutes registers the legacy trip-difficulty API routes.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.AddRouteGroup(r, baseURL, h)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		h.internalError(w, "list trip difficulties", err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.idFromRequest(w, r)
	if !ok {
		return
	}
	item, err := h.store.Get(r.Context(), id)
	if h.writeStoreError(w, "get trip difficulty", err) {
		return
	}
	helpers.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeAndValidate(w, r)
	if !ok {
		return
	}
	item, err := h.store.Create(r.Context(), input)
	if h.writeStoreError(w, "create trip difficulty", err) {
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := h.idFromRequest(w, r)
	if !ok {
		return
	}
	input, ok := h.decodeAndValidate(w, r)
	if !ok {
		return
	}
	item, err := h.store.Update(r.Context(), id, input)
	if h.writeStoreError(w, "update trip difficulty", err) {
		return
	}
	helpers.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.idFromRequest(w, r)
	if !ok {
		return
	}
	if h.writeStoreError(w, "delete trip difficulty", h.store.Delete(r.Context(), id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) idFromRequest(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := helpers.GetIDFromRequest(r)
	if err != nil || id < 1 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid record ID")
		return 0, false
	}
	return id, true
}

func (h *Handler) decodeAndValidate(w http.ResponseWriter, r *http.Request) (models.TripDifficulty, bool) {
	var input models.TripDifficulty
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request payload")
		return models.TripDifficulty{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request payload")
		return models.TripDifficulty{}, false
	}
	if err := validate(input); err != nil {
		helpers.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
		return models.TripDifficulty{}, false
	}
	input.Level = strings.TrimSpace(input.Level)
	input.LevelShort = strings.TrimSpace(input.LevelShort)
	input.Description = strings.TrimSpace(input.Description)
	return input, true
}

func validate(input models.TripDifficulty) error {
	level := strings.TrimSpace(input.Level)
	short := strings.TrimSpace(input.LevelShort)
	description := strings.TrimSpace(input.Description)
	switch {
	case level == "" || len(level) > 50:
		return fmt.Errorf("level must be between 1 and 50 characters")
	case short == "" || len(short) > 3:
		return fmt.Errorf("level_short must be between 1 and 3 characters")
	case description == "" || len(description) > 255:
		return fmt.Errorf("description must be between 1 and 255 characters")
	default:
		return nil
	}
}

func (h *Handler) writeStoreError(w http.ResponseWriter, operation string, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, tripdifficulty.ErrNotFound) {
		helpers.WriteError(w, http.StatusNotFound, "not_found", "Record not found")
		return true
	}
	h.internalError(w, operation, err)
	return true
}

func (h *Handler) internalError(w http.ResponseWriter, operation string, err error) {
	log.Printf("%s%s: %v", debugTag, operation, err)
	helpers.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
}
