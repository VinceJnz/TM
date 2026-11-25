package helpers

import (
	"api-server/v2/modelMethods/tripreport"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func ReadScheduleFilter(w http.ResponseWriter, query url.Values, defaultPeriod string) (tripreport.ScheduleFilter, bool) {
	var filter tripreport.ScheduleFilter
	for name, id := range map[string]*int{"type": &filter.TypeID, "grade": &filter.GradeID} {
		if raw := query.Get(name); raw != "" {
			n, err := strconv.ParseInt(raw, 10, 32)
			if err != nil || n <= 0 {
				WriteError(w, 400, "invalid_trip_filter", "Select a valid trip type or grade")
				return filter, false
			}
			*id = int(n)
		}
	}
	period := query.Get("month")
	if period == "" {
		period = defaultPeriod
	}
	switch period {
	case "":
	case "upcoming":
		filter.Upcoming = true
	case "past":
		filter.Past = true
	default:
		start, err := time.Parse("2006-01", period)
		if err != nil || start.Year() < 1 {
			WriteError(w, 400, "invalid_trip_filter", "Use YYYY-MM for the month")
			return filter, false
		}
		filter.Month = &start
	}
	return filter, true
}
