package handlerTrip

import (
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/tripreport"
	"api-server/v2/models"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) GetParticipantStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session)
	if !ok || session == nil || session.UserID <= 0 {
		helpers.WriteError(w, 401, "unauthorized", "Authentication required")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(session.AccessScope), "any") {
		helpers.WriteError(w, 403, "forbidden", "Global trip report requires any scope")
		return
	}
	limit, offset := 100, 0
	for name, target := range map[string]*int{"limit": &limit, "offset": &offset} {
		if raw := r.URL.Query().Get(name); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				helpers.WriteError(w, 400, "invalid_pagination", "limit must be 1..500 and offset 0..1000000")
				return
			}
			*target = n
		}
	}
	if limit < 1 || limit > 500 || offset < 0 || offset > 1000000 {
		helpers.WriteError(w, 400, "invalid_pagination", "limit must be 1..500 and offset 0..1000000")
		return
	}
	var items []models.TripParticipantStatus
	var more bool
	var err error
	store := tripreport.Store{DB: h.appConf.Db}
	switch {
	case r.URL.Query().Get("view") == "filters":
		options, err := store.Filters(r.Context())
		if err != nil {
			helpers.WriteDataError(w, err)
			return
		}
		helpers.WriteJSON(w, 200, options)
		return
	case r.URL.Query().Get("view") == "schedule":
		query := r.URL.Query()
		filter := tripreport.ScheduleFilter{ActorID: session.UserID}
		for name, id := range map[string]*int{"type": &filter.TypeID, "grade": &filter.GradeID} {
			if raw := query.Get(name); raw != "" {
				n, err := strconv.ParseInt(raw, 10, 32)
				if err != nil || n <= 0 {
					helpers.WriteError(w, 400, "invalid_trip_filter", "Select a valid trip type or grade")
					return
				}
				*id = int(n)
			}
		}
		switch month := query.Get("month"); month {
		case "past":
			filter.Past = true
		case "", "upcoming":
			filter.Upcoming = true
		default:
			start, err := time.Parse("2006-01", month)
			if err != nil || start.Year() < 1 {
				helpers.WriteError(w, 400, "invalid_trip_filter", "Use YYYY-MM for the month")
				return
			}
			filter.Month = &start
		}
		summaries, more, err := store.Schedule(r.Context(), filter, limit, offset)
		if err != nil {
			helpers.WriteDataError(w, err)
			return
		}
		writeReportPage(w, limit, offset, more, summaries)
		return
	case r.URL.Query().Get("view") == "trips":
		items, more, err = store.Trips(r.Context(), limit, offset)
	case r.URL.Query().Get("trip_id") != "":
		id, e := strconv.Atoi(r.URL.Query().Get("trip_id"))
		if e != nil || id <= 0 {
			helpers.WriteError(w, 400, "invalid_trip", "A positive trip ID is required")
			return
		}
		items, more, err = store.Participants(r.Context(), id, limit, offset)
	default:
		items, more, err = store.List(r.Context(), limit, offset)
	}
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	writeReportPage(w, limit, offset, more, items)
}

func writeReportPage(w http.ResponseWriter, limit, offset int, more bool, items any) {
	w.Header().Set("X-Page-Limit", strconv.Itoa(limit))
	if more {
		w.Header().Set("X-Next-Offset", strconv.Itoa(offset+limit))
	}
	helpers.WriteJSON(w, 200, items)
}
