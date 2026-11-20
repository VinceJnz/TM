package dbStandardTemplate

import (
	"api-server/v2/internal/testdb"
	"net/http/httptest"
	"testing"
)

func TestPagesAreBoundedAndKeepOwnershipFilter(t *testing.T) {
	db := testdb.Open(t, `CREATE TABLE items(id integer PRIMARY KEY,owner integer);INSERT INTO items SELECT n,CASE WHEN n%2=0 THEN 1 ELSE 2 END FROM generate_series(1,250)n`)
	var ids []int
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?limit=10", nil)
	if err := selectPage(w, r, db, &ids, "SELECT id FROM items WHERE owner=$1", 1); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 10 || ids[0] != 2 || ids[9] != 20 || w.Header().Get("X-Next-Offset") != "10" {
		t.Fatal(ids, w.Header())
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/?limit=10&offset=10", nil)
	if err := selectPage(w, r, db, &ids, "SELECT id FROM items WHERE owner=$1", 1); err != nil || ids[0] != 22 {
		t.Fatal(ids, err)
	}
	for _, query := range []string{"limit=0", "limit=501", "offset=-1", "offset=1%3BDROP%20TABLE%20items"} {
		if selectPage(httptest.NewRecorder(), httptest.NewRequest("GET", "/?"+query, nil), db, &ids, "SELECT id FROM items") == nil {
			t.Fatal(query)
		}
	}
}
