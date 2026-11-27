package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"testing"
)

func TestAppearanceUpgradeGrantsAdoptedLowercaseMethods(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	// Recreate the state of an adopted database immediately before migration 0037.
	if _, err := db.Exec(`
 DELETE FROM st_group_resource WHERE resource_id IN (SELECT id FROM et_resource WHERE lower(name)='siteappearance');
 DELETE FROM et_resource WHERE lower(name)='siteappearance';
 DROP TABLE st_appearance_revision;
 DELETE FROM schema_migrations WHERE version='0037_site_appearance.sql';
 UPDATE et_access_level SET name=lower(name);
 `); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	var grants []struct {
		GroupID int    `db:"group_id"`
		Method  string `db:"method"`
	}
	err := db.Select(&grants, `
 SELECT gr.group_id, lower(method.name) AS method
 FROM st_group_resource gr
 JOIN et_resource resource ON resource.id=gr.resource_id
 JOIN et_access_level method ON method.id=gr.access_level_id
 JOIN et_access_scope scope ON scope.id=gr.access_scope_id
 WHERE lower(resource.name)='siteappearance' AND lower(scope.name)='any'
 ORDER BY gr.group_id, method`)
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 2 || grants[0].GroupID != 3 || grants[0].Method != "get" || grants[1].GroupID != 3 || grants[1].Method != "put" {
		t.Fatalf("expected only the security administrator's read and write grants, got %+v", grants)
	}
}
