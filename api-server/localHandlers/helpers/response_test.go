package helpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteJSON(recorder, http.StatusCreated, map[string]int{"id": 7})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var response map[string]int
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["id"] != 7 {
		t.Fatalf("id = %d, want 7", response["id"])
	}
}

func TestWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteError(recorder, http.StatusBadRequest, "invalid_request", "Invalid request payload")

	var response ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "invalid_request" || response.Error.Message != "Invalid request payload" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
