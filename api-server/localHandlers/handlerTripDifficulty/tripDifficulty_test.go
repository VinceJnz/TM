package handlerTripDifficulty

import (
	"api-server/v2/modelMethods/tripdifficulty"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

type fakeStore struct {
	getItem    models.TripDifficulty
	getErr     error
	createItem models.TripDifficulty
}

func (s fakeStore) List(context.Context) ([]models.TripDifficulty, error) { return nil, nil }
func (s fakeStore) Get(context.Context, int) (models.TripDifficulty, error) {
	return s.getItem, s.getErr
}
func (s fakeStore) Create(context.Context, models.TripDifficulty) (models.TripDifficulty, error) {
	return s.createItem, nil
}
func (s fakeStore) Update(context.Context, int, models.TripDifficulty) (models.TripDifficulty, error) {
	return models.TripDifficulty{}, nil
}
func (s fakeStore) Delete(context.Context, int) error { return nil }

func TestGetRejectsInvalidID(t *testing.T) {
	handler := &Handler{store: fakeStore{}}
	request := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/tripDifficulty/0", nil), map[string]string{"id": "0"})
	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

	assertError(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestCreateRejectsMalformedJSON(t *testing.T) {
	handler := &Handler{store: fakeStore{}}
	recorder := httptest.NewRecorder()

	handler.Create(recorder, httptest.NewRequest(http.MethodPost, "/tripDifficulty", strings.NewReader(`{"level":`)))

	assertError(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestCreateRejectsInvalidValues(t *testing.T) {
	handler := &Handler{store: fakeStore{}}
	recorder := httptest.NewRecorder()

	handler.Create(recorder, httptest.NewRequest(http.MethodPost, "/tripDifficulty", strings.NewReader(`{"level":"","level_short":"TOOLONG","description":""}`)))

	assertError(t, recorder, http.StatusUnprocessableEntity, "validation_failed")
}

func TestGetMapsMissingRecordToNotFound(t *testing.T) {
	handler := &Handler{store: fakeStore{getErr: tripdifficulty.ErrNotFound}}
	request := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/tripDifficulty/7", nil), map[string]string{"id": "7"})
	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

	assertError(t, recorder, http.StatusNotFound, "not_found")
}

func TestCreateReturnsCreatedRecord(t *testing.T) {
	expected := models.TripDifficulty{ID: 7, Level: "Easy", LevelShort: "E", Description: "Suitable for beginners"}
	handler := &Handler{store: fakeStore{createItem: expected}}
	recorder := httptest.NewRecorder()

	handler.Create(recorder, httptest.NewRequest(http.MethodPost, "/tripDifficulty", strings.NewReader(`{"level":"Easy","level_short":"E","description":"Suitable for beginners"}`)))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	var actual models.TripDifficulty
	if err := json.NewDecoder(recorder.Body).Decode(&actual); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if actual.ID != expected.ID || actual.Level != expected.Level || actual.LevelShort != expected.LevelShort {
		t.Fatalf("response = %+v, want %+v", actual, expected)
	}
}

func assertError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, status, recorder.Body.String())
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != code {
		t.Fatalf("error code = %q, want %q", response.Error.Code, code)
	}
}
