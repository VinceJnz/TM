package handlerAuth

import (
	"encoding/json"
	"log"
	"net/http"
)

func writeJSONResponse(w http.ResponseWriter, statusCode int, payload any, context string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("%s%s failed to encode JSON response: %v", debugTag, context, err)
	}
}
