package handlerAuth

import (
	"net/http"

	"api-server/v2/localHandlers/helpers"
)

func writeJSONResponse(w http.ResponseWriter, statusCode int, payload any, context string) {
	if statusCode >= 400 {
		code, message := "request_failed", "Request failed"
		switch v := payload.(type) {
		case map[string]string:
			if v["error"] != "" {
				code = v["error"]
			}
			if v["message"] != "" {
				message = v["message"]
			}
		case map[string]any:
			if s, ok := v["error"].(string); ok {
				code = s
			}
			if s, ok := v["message"].(string); ok {
				message = s
			}
		default:
			helpers.WriteJSON(w, statusCode, payload)
			return
		}
		helpers.WriteError(w, statusCode, code, message)
		return
	}
	helpers.WriteJSON(w, statusCode, payload)
}
