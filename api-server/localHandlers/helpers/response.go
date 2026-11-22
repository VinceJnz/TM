package helpers

import (
	"api-server/v2/models"
	"net/http"
)

// ErrorResponse is the stable JSON shape returned for API errors.
// Code is intended for clients; Message is safe to display to users.
type ErrorResponse = models.ErrorResponse

type APIError = models.APIError

// WriteError writes an API error without exposing internal error details.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Error: APIError{Code: code, Message: message}})
}
