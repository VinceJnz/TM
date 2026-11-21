package helpers

import (
	"net/http"
	"strconv"
)

// OptionalQueryID distinguishes an omitted parent filter from malformed input.
func OptionalQueryID(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	values, present := r.URL.Query()[name]
	if !present {
		return 0, true
	}
	if len(values) != 1 {
		WriteError(w, 400, "invalid_id", "Invalid parent ID")
		return 0, false
	}
	id, err := strconv.Atoi(values[0])
	if err != nil || id <= 0 {
		WriteError(w, 400, "invalid_id", "Invalid parent ID")
		return 0, false
	}
	return id, true
}
