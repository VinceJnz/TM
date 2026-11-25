package handlerAuth

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPendingRegistrationHashesPasswordBeforePersistence(t *testing.T) {
	const password = "registration-test-password"
	var request registrationRequest
	// Public JSON keeps its existing field names; administrative fields are ignored.
	if err := json.Unmarshal([]byte(`{"username":"test-user","name":"Test User","email":"user@example.test","user_password":"`+password+`","id":99,"provider":"google","user_account_status_id":2,"user_account_hidden":true}`), &request); err != nil {
		t.Fatal(err)
	}
	data, err := encodePendingRegistration(request)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(password)) || bytes.Contains(data, []byte(`"user_password"`)) {
		t.Fatal("pending registration contains plaintext password or public password field")
	}
	var pending pendingRegistration
	if err := json.Unmarshal(data, &pending); err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pending.PasswordHash), []byte(password)); err != nil {
		t.Fatal("pending registration does not contain a usable password hash")
	}
	user, err := decodePendingRegistration(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if user.Password.String != pending.PasswordHash {
		t.Fatal("verification changed the stored hash (would break password login)")
	}
	if user.Username != request.Username || user.Name != request.Name || user.Email != request.Email {
		t.Fatal("registration profile did not survive persistence")
	}
	if user.ID != 0 || user.Provider.Valid || user.AccountStatusID.Valid {
		t.Fatal("public registration populated administrative fields")
	}
	if !user.AccountHidden.Valid || !user.AccountHidden.Bool {
		t.Fatal("registration lost the checked privacy preference")
	}
}

func TestPendingRegistrationUncheckedPrivacyPreference(t *testing.T) {
	hidden := false
	request := registrationRequest{}
	request.AccountHidden = &hidden
	data, err := encodePendingRegistration(request)
	if err != nil {
		t.Fatal(err)
	}
	user, err := decodePendingRegistration(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if !user.AccountHidden.Valid || user.AccountHidden.Bool {
		t.Fatal("registration lost the unchecked privacy preference")
	}
}

func TestPendingRegistrationWithoutPassword(t *testing.T) {
	data, err := encodePendingRegistration(registrationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	user, err := decodePendingRegistration(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if user.Password.Valid || bytes.Contains(data, []byte("password")) {
		t.Fatal("passwordless registration acquired a password")
	}
}

func TestPendingRegistrationRejectsInvalidPasswordLengths(t *testing.T) {
	for _, password := range []string{"short", strings.Repeat("x", 73), strings.Repeat("é", 37)} {
		var request registrationRequest
		request.Password.SetValid(password)
		if _, err := encodePendingRegistration(request); err == nil {
			t.Fatal("accepted password outside bcrypt byte limits")
		}
	}
}

func TestPendingRegistrationRejectsLegacyAndCorruptData(t *testing.T) {
	for _, data := range []string{
		`{"username":"legacy","user_password":"legacy-plaintext"}`,
		`{"version":2}`,
		`{"version":1,"password_hash":"not-a-hash"}`,
		`{`,
	} {
		if _, err := decodePendingRegistration(data); err == nil {
			t.Fatal("accepted legacy or corrupt pending registration")
		}
	}
}

func TestRegisterInvalidPasswordDoesNotLogCredentials(t *testing.T) {
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousOutput) })
	// Invalid input must be rejected before database access, without logging payloads.
	h := &Handler{}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"username":"test-user","email":"user@example.test","user_password":"short"}`))
	response := httptest.NewRecorder()
	h.Register(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
	if strings.Contains(logs.String(), "short") || strings.Contains(logs.String(), "user@example.test") {
		t.Fatal("registration logged its payload")
	}
}
