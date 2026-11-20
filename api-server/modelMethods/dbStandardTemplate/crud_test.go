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
