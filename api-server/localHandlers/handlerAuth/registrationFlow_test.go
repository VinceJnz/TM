package handlerAuth

import (
	"api-server/v2/app/appCore"
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// A small database/sql test connection exercises the actual registration handlers
// and persistence helpers without a live database or an additional dependency.
type registrationDB struct {
	data         string
	token        string
	passwordHash string
	failSave     bool
	tokenDeleted bool
	userCreated  bool
}

func (db *registrationDB) Connect(context.Context) (driver.Conn, error) { return db, nil }
func (db *registrationDB) Driver() driver.Driver                        { return db }
func (db *registrationDB) Open(string) (driver.Conn, error)             { return db, nil }
func (db *registrationDB) Close() error                                 { return nil }
func (db *registrationDB) Begin() (driver.Tx, error)                    { return registrationTx{}, nil }
func (db *registrationDB) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepared statement")
}

type registrationTx struct{}

func (registrationTx) Commit() error   { return nil }
func (registrationTx) Rollback() error { return nil }

type registrationRows struct {
	columns []string
	values  []driver.Value
}

func (r *registrationRows) Columns() []string { return r.columns }
func (r *registrationRows) Close() error      { return nil }
func (r *registrationRows) Next(dest []driver.Value) error {
	if r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.values = nil
	return nil
}

func (db *registrationDB) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "SELECT age_group"):
		return &registrationRows{[]string{"age_group"}, []driver.Value{"adult"}}, nil
	case strings.HasPrefix(query, "INSERT INTO st_token"):
		db.token = args[3].Value.(string)
		return &registrationRows{[]string{"id"}, []driver.Value{int64(1)}}, nil
	case strings.HasPrefix(query, "SELECT ID FROM st_token"):
		if args[0].Value == int64(1) {
			return &registrationRows{[]string{"id"}, []driver.Value{int64(1)}}, nil
		}
		return &registrationRows{columns: []string{"id"}}, nil
	case strings.Contains(query, "SELECT stt.ID"), strings.HasPrefix(query, "DELETE FROM st_token WHERE name"):
		if strings.HasPrefix(query, "DELETE") {
			db.tokenDeleted = true
		}
		return &registrationRows{
			[]string{"id", "user_id", "name", "token", "session_data", "token_valid", "valid_from", "valid_to"},
			[]driver.Value{int64(1), int64(0), "registration-verification", db.token, db.data, true, time.Now().Add(-time.Minute), time.Now().Add(time.Hour)},
		}, nil
	case strings.HasPrefix(query, "INSERT INTO st_users"):
		db.passwordHash, _ = args[7].Value.(string)
		db.userCreated = true
		return &registrationRows{[]string{"id"}, []driver.Value{int64(42)}}, nil
	case strings.Contains(query, "FROM st_users"):
		return &registrationRows{columns: []string{"id"}}, nil
	default:
		return nil, errors.New("unexpected registration query")
	}
}

func (db *registrationDB) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if strings.HasPrefix(query, "UPDATE st_token") {
		if db.failSave {
			return nil, errors.New("simulated pending registration write failure")
		}
		db.data = args[4].Value.(string)
	} else if strings.HasPrefix(query, "DELETE FROM st_token WHERE") {
		db.tokenDeleted = true
	} else if !strings.HasPrefix(query, "DELETE FROM st_token") {
		return nil, errors.New("unexpected registration command")
	}
	return driver.RowsAffected(1), nil
}

func registrationHandler(t *testing.T, db *registrationDB) (*Handler, *bytes.Buffer) {
	t.Helper()
	connection := sql.OpenDB(db)
	t.Cleanup(func() { connection.Close() })
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousOutput) })
	return New(&appCore.Config{Db: sqlx.NewDb(connection, "postgres"), EmailSvc: testMailer{}}), &logs
}

func TestRegistrationFlowPersistsOnlyHashAndDoesNotLogSecrets(t *testing.T) {
	db := &registrationDB{}
	h, logs := registrationHandler(t, db)
	const password = "registration-flow-password"
	response := httptest.NewRecorder()
	h.Register(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(
		`{"username":"test-user","name":"Test User","email":"user@example.test","user_password":"`+password+`","user_birth_date":"1990-01-01T00:00:00Z","user_age_group_id":1}`)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("register returned %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(db.data, password) {
		t.Fatal("database received plaintext registration password")
	}
	user, err := decodePendingRegistration(db.data)
	if err != nil {
		t.Fatal(err)
	}
	hash := user.Password.String
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatal("persisted hash cannot verify original password")
	}
	response = httptest.NewRecorder()
	h.VerifyRegistration(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-registration", strings.NewReader(`{"token":"`+db.token+`"}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("verify returned %d: %s", response.Code, response.Body.String())
	}
	if !db.userCreated || db.passwordHash != hash || !db.tokenDeleted {
		t.Fatal("verification did not create user with original hash and consume token")
	}
	for _, secret := range []string{password, hash, db.token} {
		if secret == "" || strings.Contains(logs.String(), secret) || strings.Contains(response.Body.String(), secret) {
			t.Fatal("registration exposed credentials or verification token")
		}
	}
}

func TestRegistrationStopsWhenPendingDataCannotBeSaved(t *testing.T) {
	db := &registrationDB{failSave: true}
	h, _ := registrationHandler(t, db)
	response := httptest.NewRecorder()
	h.Register(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(
		`{"username":"test-user","name":"Test User","email":"user@example.test","user_birth_date":"1990-01-01T00:00:00Z","user_age_group_id":1}`)))
	if response.Code != http.StatusInternalServerError || !db.tokenDeleted || db.userCreated {
		t.Fatal("failed pending-data write was not rejected and cleaned up")
	}
}

func TestVerifyRegistrationRejectsLegacyPlaintextBeforeUserCreation(t *testing.T) {
	db := &registrationDB{token: "legacy-verification-token", data: `{"username":"legacy","user_password":"legacy-password"}`}
	h, logs := registrationHandler(t, db)
	response := httptest.NewRecorder()
	h.VerifyRegistration(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-registration", strings.NewReader(`{"token":"legacy-verification-token"}`)))
	if response.Code != http.StatusBadRequest || db.userCreated {
		t.Fatal("legacy plaintext registration was accepted")
	}
	for _, secret := range []string{"legacy-password", db.token} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("legacy registration credentials appeared in logs")
		}
	}
}

type testMailer struct{}

func (testMailer) SendMail(string, string, string) (bool, error) { return true, nil }
func (testMailer) RenewURL() string                              { return "" }
