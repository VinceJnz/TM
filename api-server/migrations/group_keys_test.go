package migrations

import (
	"context"
	"testing"

	"api-server/v2/internal/testdb"
)

func TestBuiltInGroupKeys(t *testing.T) {
	db := testdb.Open(t, "")
	if err := Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE st_group SET name='Configuration team' WHERE system_key='sysadmin'`); err != nil {
		t.Fatal(err)
	}
	var key string
	if err := db.Get(&key, `SELECT system_key FROM st_group WHERE name='Configuration team'`); err != nil || key != "sysadmin" {
		t.Fatalf("renamed group key = %q, error = %v", key, err)
	}
	for _, query := range []string{
		`UPDATE st_group SET system_key=NULL WHERE system_key='sysadmin'`,
		`DELETE FROM st_group WHERE system_key='sysadmin'`,
		`INSERT INTO st_group(name,system_key) VALUES('Custom','custom')`,
	} {
		if _, err := db.Exec(query); err == nil {
			t.Fatalf("expected rejection for %s", query)
		}
	}
}
