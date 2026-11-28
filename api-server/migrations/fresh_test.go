package migrations

import (
	"context"
	"reflect"
	"testing"

	"api-server/v2/internal/testdb"
)

func TestFreshBaseline(t *testing.T) {
	ctx := context.Background()
	fresh := testdb.Open(t, "")
	if err := ApplyFresh(ctx, fresh); err != nil {
		t.Fatal(err)
	}
	if err := Check(ctx, fresh); err != nil {
		t.Fatal(err)
	}
	if err := ApplyFresh(ctx, fresh); err == nil {
		t.Fatal("expected rejection on a populated schema")
	}
	chain := testdb.Open(t, "")
	if err := Apply(ctx, chain, false); err != nil {
		t.Fatal(err)
	}
	query, err := catalogFiles.ReadFile("catalog.query")
	if err != nil {
		t.Fatal(err)
	}
	structure := "SELECT * FROM (" + string(query) + ") AS snapshot WHERE kind <> 'grant' ORDER BY kind,name"
	var want, got []CatalogEntry
	if err := chain.Select(&want, structure); err != nil {
		t.Fatal(err)
	}
	if err := fresh.Select(&got, structure); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("consolidated schema differs from the migration chain")
	}
	var admin, sysadmin int
	if err := fresh.QueryRow(`
		SELECT count(*) FILTER (WHERE system_key='admin'), count(*) FILTER (WHERE system_key='sysadmin')
		FROM st_group_resource r JOIN st_group g ON g.id=r.group_id
	`).Scan(&admin, &sysadmin); err != nil {
		t.Fatal(err)
	}
	if admin != 22 || sysadmin != 108 {
		t.Fatalf("unexpected administrative grant counts: Admin %d, SysAdmin %d", admin, sysadmin)
	}
	for _, db := range []catalogReader{chain, fresh} {
		var keys []string
		if err := db.SelectContext(ctx, &keys, `
			SELECT DISTINCT g.system_key FROM st_group_resource gr
			JOIN st_group g ON g.id=gr.group_id JOIN et_resource r ON r.id=gr.resource_id
			WHERE lower(r.name)='bookingstatus' ORDER BY g.system_key
		`); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(keys, []string{"sysadmin"}) {
			t.Fatalf("unexpected Booking Status grant groups: %v", keys)
		}
	}
}
