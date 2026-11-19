// Package testdb supplies isolated schemas for PostgreSQL integration tests.
package testdb

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Open(t *testing.T, schemaSQL string) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TM_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TM_TEST_DATABASE_URL is required for disposable PostgreSQL integration tests")
	}
	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "tm_test_" + hex.EncodeToString(random[:])
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sqlx.Connect("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	return db
}
