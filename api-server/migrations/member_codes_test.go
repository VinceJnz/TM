package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"fmt"
	"testing"
)

func TestMemberCodesAllocatedAndUniqueWithoutRewritingIdentities(t *testing.T) {
	db := testdb.Open(t, "")
	if err := Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(100,'Synthetic','synthetic','synthetic@example.test'),(101,'Other','other','other@example.test')`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.Get(&count, `SELECT count(DISTINCT member_code) FROM st_users WHERE length(member_code)>0`); err != nil || count != 2 {
		t.Fatal(count, err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email,member_code) SELECT 102,'Duplicate','duplicate','duplicate@example.test',' '||lower(member_code)||' ' FROM st_users WHERE id=100`); err == nil {
		t.Fatal("duplicate identity accepted")
	}
}

func TestMemberCodesPreserveLargeAccountIDs(t *testing.T) {
	db := testdb.Open(t, "")
	if err := Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	for id := 1048576; id < 1048593; id++ {
		if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES($1,'Synthetic',$2::text,$2::text||'@example.test')`, id, fmt.Sprintf("large-%d", id)); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT count(DISTINCT member_code) FROM st_users WHERE id>=1048576`); err != nil || count != 17 {
		t.Fatal(count, err)
	}
}
