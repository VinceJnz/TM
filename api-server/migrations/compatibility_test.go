package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"testing"
)

func TestCompatibilityDetectsSchemaAndSeedDriftWithoutWriting(t *testing.T) {
	baseline, err := files.ReadFile("0001_baseline.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t, string(baseline))
	ctx := context.Background()
	differences, err := Compatibility(ctx, db)
	if err != nil || len(differences) != 0 {
		t.Fatal(differences, err)
	}
	if _, err = db.Exec(`ALTER TABLE st_users DROP CONSTRAINT users_username_unique;DROP INDEX booking_owner;ALTER TABLE st_users ALTER COLUMN username TYPE varchar(300);UPDATE et_access_scope SET name='changed' WHERE lower(name)='own'`); err != nil {
		t.Fatal(err)
	}
	differences, err = Compatibility(ctx, db)
	if err != nil || len(differences) < 4 {
		t.Fatal(differences, err)
	}
	var history bool
	if err = db.Get(&history, "SELECT to_regclass('schema_migrations') IS NOT NULL"); err != nil || history {
		t.Fatal("read-only report changed migration history", err)
	}
	if err = Apply(ctx, db, true); err == nil {
		t.Fatal("incompatible schema adopted")
	}
}
