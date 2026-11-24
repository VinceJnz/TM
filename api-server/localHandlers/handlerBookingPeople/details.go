package handlerBookingPeople

import (
	"api-server/v2/localHandlers/helpers"
	"net/http"
)

func (h *Handler) getQuestionnaire(w http.ResponseWriter, r *http.Request) {
	session := h.session(w, r)
	if session == nil {
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	result, err := h.store.Questionnaire(r.Context(), id, session.UserID, session.AccessScope)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteJSON(w, 200, result)
}

func (h *Handler) getDetails(w http.ResponseWriter, r *http.Request) {
	h.readDetails(w, r, false)
}

func (h *Handler) getLeaderDetails(w http.ResponseWriter, r *http.Request) {
	h.readDetails(w, r, true)
}

func (h *Handler) readDetails(w http.ResponseWriter, r *http.Request, leader bool) {
	session := h.session(w, r)
	if session == nil {
		return
	}
	if (leader && session.AccessScope != "any") ||
		(!leader && session.AccessScope != "own" && session.AccessScope != "any") {
		helpers.WriteError(w, 403, "forbidden", "Participant access required")
		return
	}
	id := helpers.RouteID(w, r)
	if id <= 0 {
		return
	}
	result, err := h.store.Details(r.Context(), id, session.UserID, session.AccessScope, leader)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	helpers.WriteJSON(w, 200, result)
}
