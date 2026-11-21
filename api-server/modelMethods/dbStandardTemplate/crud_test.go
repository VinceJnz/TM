package dbStandardTemplate

import (
	"api-server/v2/internal/testdb"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"testing"
)

func TestInvalidIDStopsBeforeDatabase(t *testing.T) {
	rc := NewResourceCRUD(ResourceCRUDConfig[struct{}]{})
	for _, id := range []string{"bad", "0", "-1", "99999999999999999999999999999999"} {
		r := mux.SetURLVars(httptest.NewRequest("DELETE", "/", nil), map[string]string{"id": id})
		w := httptest.NewRecorder()
		rc.Delete(w, r)
		if w.Code != 400 {
			t.Fatal(id, w.Code)
		}
	}
}

func TestSingleRecordScansStructAndPreservesArrayContract(t *testing.T) {
	db := testdb.Open(t, "CREATE TABLE items(id integer PRIMARY KEY,value text);INSERT INTO items VALUES(1,'original')")
	type item struct {
		ID    int    `db:"id" json:"id"`
		Value string `db:"value" json:"value"`
	}
	for _, id := range []int{1, 99} {
		records := []item{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Get(w, r, "test", db, &records, "SELECT id,value FROM items WHERE id=$1", id)
		if id == 99 {
			if w.Code != 404 {
				t.Fatal(w.Code)
			}
			continue
		}
		if w.Code != 200 || len(records) != 1 || records[0].Value != "original" || w.Body.String()[0] != '[' {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestMissingWritesReturnNotFound(t *testing.T) {
	db := testdb.Open(t, "CREATE TABLE items(id integer PRIMARY KEY,value text);INSERT INTO items VALUES(1,'original')")
	for _, operation := range []string{"update", "delete"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("PUT", "/", nil)
		if operation == "update" {
			Update(w, r, "test", db, nil, "UPDATE items SET value='changed' WHERE id=$1", 2)
		} else {
			Delete(w, r, "test", db, nil, "DELETE FROM items WHERE id=$1", 2)
		}
		if w.Code != 404 {
			t.Fatal(operation, w.Code)
		}
	}
}
