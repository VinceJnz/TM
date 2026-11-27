package handlerAuth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func issueTestSession(t *testing.T, h *Handler) *http.Cookie {
	t.Helper()
	raw, err := h.authentication().Issue(context.Background(), 1, "localhost", "session", time.Now().Add(time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "session", Value: raw}
}
