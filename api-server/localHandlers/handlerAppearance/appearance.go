package handlerAppearance

import (
	"api-server/v2/app/appCore"
	theme "api-server/v2/app/services/appearance"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/appearance"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type handler struct {
	app   *appCore.Config
	store appearance.Store
}

type settings struct {
	Version    int64       `json:"version"`
	Theme      theme.Theme `json:"theme"`
	CanRestore bool        `json:"canRestore"`
	Action     string      `json:"action,omitempty"`
}

func Register(app *appCore.Config, public, protected *mux.Router) {
	h := handler{app: app, store: appearance.Store{DB: app.Db}}
	public.HandleFunc("/site-theme.css", h.css).Methods("GET")
	public.HandleFunc("/site-brand.js", h.brand).Methods("GET")
	public.HandleFunc("/site-logo", h.asset(true)).Methods("GET")
	public.HandleFunc("/site-icon", h.asset(false)).Methods("GET")
	helpers.CapabilityRoute(protected, "/siteAppearance", "siteAppearance", h.admin).Methods("GET", "PUT")
}

func (h handler) read(r *http.Request, previous bool) (settings, error) {
	revision, err := h.store.Read(r.Context(), previous)
	if err != nil {
		return settings{}, err
	}
	result := settings{Version: revision.Version, CanRestore: revision.Version > 1}
	err = json.Unmarshal(revision.Settings, &result.Theme)
	return result, err
}

func publicType(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
}

func (h handler) css(w http.ResponseWriter, r *http.Request) {
	publicType(w, "text/css; charset=utf-8")
	if current, err := h.read(r, false); err == nil && current.Theme != (theme.Theme{}) {
		io.WriteString(w, current.Theme.CSS())
	}
}

func (h handler) brand(w http.ResponseWriter, r *http.Request) {
	publicType(w, "application/javascript; charset=utf-8")
	current, _ := h.read(r, false)
	data, _ := json.Marshal(struct {
		Name    string `json:"name"`
		Version int64  `json:"version"`
		Logo    bool   `json:"logo"`
	}{current.Theme.WithDefaults().Name, current.Version, current.Theme.Logo != ""})
	io.WriteString(w, "window.TM_BRAND="+string(data)+";")
}

func (h handler) asset(logo bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, _ := h.read(r, false)
		encoded, maxBytes, maxSide := current.Theme.Icon, 32*1024, 256
		if logo {
			encoded, maxBytes, maxSide = current.Theme.Logo, 256*1024, 1024
		}
		data, kind, err := theme.Image(encoded, maxBytes, maxSide)
		if !logo && len(data) == 0 {
			publicType(w, "image/x-icon")
			http.ServeFile(w, r, "./static/favicon.ico")
			return
		}
		if err != nil || len(data) == 0 {
			http.NotFound(w, r)
			return
		}
		publicType(w, kind)
		w.Write(data)
	}
}

func (h handler) admin(w http.ResponseWriter, r *http.Request) {
	session := helpers.RequiredSession(w, r, h.app.SessionIDKey)
	if session == nil {
		return
	}
	if !strings.EqualFold(session.AccessScope, "any") {
		helpers.WriteError(w, 403, "forbidden", "Appearance administration requires any scope")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		current, err := h.read(r, false)
		if err != nil {
			helpers.WriteError(w, 500, "appearance_unavailable", "Appearance settings could not be loaded")
			return
		}
		current.Theme = current.Theme.WithDefaults()
		helpers.WriteJSON(w, 200, current)
		return
	}
	var input settings
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 410*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(new(any)) != io.EOF {
		helpers.WriteError(w, 400, "invalid_appearance", "Invalid appearance request")
		return
	}
	switch input.Action {
	case "restore":
		previous, err := h.read(r, true)
		if errors.Is(err, sql.ErrNoRows) {
			helpers.WriteError(w, 409, "no_previous_theme", "There is no previous theme")
			return
		}
		if err != nil {
			helpers.WriteError(w, 500, "appearance_unavailable", "Previous appearance could not be loaded")
			return
		}
		input.Theme = previous.Theme
	case "defaults":
		input.Theme = theme.Theme{}
	case "publish":
		input.Theme = input.Theme.WithDefaults()
	default:
		helpers.WriteError(w, 400, "invalid_appearance", "Choose a publish action")
		return
	}
	if err := input.Theme.Validate(); err != nil {
		helpers.WriteError(w, 400, "invalid_appearance", err.Error())
		return
	}
	data, _ := json.Marshal(input.Theme)
	version, err := h.store.Publish(r.Context(), input.Version, data, session.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		helpers.WriteError(w, 409, "stale_record", "Appearance changed. Reload the settings before publishing.")
		return
	}
	if err != nil {
		helpers.WriteError(w, 500, "appearance_unavailable", "Appearance could not be published")
		return
	}
	helpers.WriteJSON(w, 200, settings{Version: version, Theme: input.Theme.WithDefaults(), CanRestore: true})
}
