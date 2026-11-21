package handlerAuth

import "strings"

// Only handlers with record-level ownership enforcement may receive own grants.
// Global dictionaries and administration require an explicit any-scope grant;
// an unsupported scope must not silently turn into unrestricted access.
func supportsScope(resource, method, scope string) bool {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "any":
		return true
	case "own":
		resource = strings.ToLower(resource)
		method = strings.ToUpper(method)
		switch resource {
		case "bookings", "mybookings", "bookingpeople", "myprofile":
			return method == "GET" || method == "POST" || method == "PUT" || method == "DELETE"
		case "auth":
			return method == "GET"
		case "trips":
			return method == "POST" || method == "PUT" || method == "DELETE"
		}
	}
	return false
}
