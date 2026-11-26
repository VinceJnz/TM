package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"errors"
	"testing"

	"github.com/lib/pq"
)

func TestUsernameUniquenessIgnoresCapitalization(t *testing.T) {
	db := testdb.Open(t, "")
	if err := Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users (name, username, email) VALUES
		('First','TestMember','first@example.test'), ('Second','OtherMember','second@example.test')`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO st_users (name, username, email) VALUES ('Third','testmember','third@example.test')`,
		`UPDATE st_users SET username='TESTMEMBER' WHERE username='OtherMember'`,
	} {
		_, err := db.Exec(statement)
		var conflict *pq.Error
		if !errors.As(err, &conflict) || conflict.Code != "23505" || conflict.Constraint != "users_username_case_insensitive_unique" {
			t.Fatalf("expected case-insensitive username conflict, got %v", err)
		}
	}
	if _, err := db.Exec(`UPDATE st_users SET username='testMember' WHERE username='TestMember'`); err != nil {
		t.Fatal(err)
	}
}
