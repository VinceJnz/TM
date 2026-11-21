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

func TestCursorKeepsOwnershipAndDoesNotSkipAfterEarlierDeletion(t *testing.T) {
	db := testdb.Open(t, `CREATE TABLE items(id integer PRIMARY KEY,owner integer);INSERT INTO items VALUES(1,1),(2,2),(3,1),(4,1),(5,1);`)
	var ids []int
	w := httptest.NewRecorder()
	if err := selectPage(w, httptest.NewRequest("GET", "/?pagination=cursor&limit=2", nil), db, &ids, "SELECT id FROM items WHERE owner=$1", 1); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[1] != 3 || w.Header().Get("X-Next-Cursor") != "3" {
		t.Fatal(ids, w.Header())
	}
	if _, err := db.Exec(`DELETE FROM items WHERE id=1;INSERT INTO items VALUES(6,2);`); err != nil {
		t.Fatal(err)
	}
	if err := selectPage(httptest.NewRecorder(), httptest.NewRequest("GET", "/?pagination=cursor&limit=2&after=3&offset=2", nil), db, &ids, "SELECT id FROM items WHERE owner=$1", 1); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 4 || ids[1] != 5 {
		t.Fatal("cursor skipped records or leaked another owner", ids)
	}
	for _, after := range []string{"-1", "invalid", "9223372036854775808"} {
		if err := selectPage(httptest.NewRecorder(), httptest.NewRequest("GET", "/?pagination=cursor&after="+after, nil), db, &ids, "SELECT id FROM items"); err == nil {
			t.Fatal("invalid cursor accepted", after)
		}
	}
}
