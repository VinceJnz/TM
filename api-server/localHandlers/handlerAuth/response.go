package handlerAuth

import (
	"net/http"

	"api-server/v2/localHandlers/helpers"
)

func writeJSONResponse(w http.ResponseWriter, statusCode int, payload any, context string) {
	helpers.WriteJSON(w, statusCode, payload)
}
