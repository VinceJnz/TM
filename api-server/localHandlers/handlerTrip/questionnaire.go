package handlerTrip

import (
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/trips"
	"api-server/v2/models"
	"net/http"
)

func (h *Handler) getQuestionnaire(w http.ResponseWriter, r *http.Request) {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	owner := session.UserID
	switch session.AccessScope {
	case "any":
		owner = 0
	case "own":
	default:
		helpers.WriteError(w, 403, "forbidden", "Trip access required")
		return
	}
	result, err := trips.Questionnaire(r.Context(), h.store.DB, id, owner)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, result)
}

func (h *Handler) publishQuestionnaire(w http.ResponseWriter, r *http.Request) {
	session := helpers.RequiredSession(w, r, h.appConf.SessionIDKey)
	if session == nil {
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	if session.AccessScope != "own" && session.AccessScope != "any" {
		helpers.WriteError(w, 403, "forbidden", "Trip editing permission required")
		return
	}
	var input models.Questionnaire
	if err := helpers.DecodeJSONBody(r, &input); err != nil || input.Validate() != nil {
		helpers.WriteError(w, 422, "invalid_questionnaire", "Check the question labels, formats, keys, and choices")
		return
	}
	result, err := h.store.PublishQuestionnaire(r.Context(), id, session.UserID, session.AccessScope == "any", input)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, result)
}
