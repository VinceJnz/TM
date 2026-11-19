package handlerAuth

import (
	"api-server/v2/app/appCore"
	"net/http/httptest"
	"testing"
)

func TestMissingEmailReturnsUnavailable(t *testing.T) {
	h := New(&appCore.Config{})
	w := httptest.NewRecorder()
	h.LoginSendOTP(w, httptest.NewRequest("POST", "/login", nil))
	if w.Code != 503 {
		t.Fatal("missing email accepted authentication request")
	}
}
