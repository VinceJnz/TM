package httpProcessor

import (
	"encoding/json"
	"fmt"
	"strings"
)

type APIError struct {
	Status        int
	Code, Message string
}

func (e *APIError) Error() string { return e.Message }
func responseError(status int, body []byte) error {
	var v struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &v) == nil {
		var nested struct{ Code, Message string }
		if json.Unmarshal(v.Error, &nested) == nil && nested.Message != "" {
			return &APIError{status, nested.Code, nested.Message}
		}
		var code string
		if json.Unmarshal(v.Error, &code) == nil {
			message := v.Message
			if message == "" {
				message = code
			}
			return &APIError{status, code, message}
		}
	}
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = fmt.Sprintf("Request failed (%d)", status)
	}
	return &APIError{status, "request_failed", message}
}
