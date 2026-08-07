package helpers

import "net/http"

// ErrorResponse is the stable JSON shape returned for API errors.
// Code is intended for clients; Message is safe to display to users.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteError writes an API error without exposing internal error details.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Error: APIError{Code: code, Message: message}})
}
