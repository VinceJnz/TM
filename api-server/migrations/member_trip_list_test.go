package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"testing"
)

func TestMemberTripListGrantUpgradeIsReadOnlyAndRepeatable(t *testing.T) {
	db := testdb.Open(t, "")
	if err := Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	// Reproduce an adopted installation with the old own-report grant.
	if _, err := db.Exec(`DELETE FROM st_group_resource WHERE group_id=1 AND resource_id=(SELECT id FROM et_resource WHERE name='tripsReport');
 INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id)
 SELECT 1,r.id,l.id,s.id FROM et_resource r,et_access_level l,et_access_scope s WHERE r.name='tripsReport' AND l.name='GET' AND s.name='own';`); err != nil {
		t.Fatal(err)
	}
	var before string
	const untouched = `SELECT COALESCE(string_agg(row(group_id,resource_id,access_level_id,access_scope_id)::text,',' ORDER BY group_id,resource_id,access_level_id,access_scope_id),'') FROM st_group_resource WHERE NOT(group_id=1 AND resource_id=(SELECT id FROM et_resource WHERE name='tripsReport'))`
	if err := db.Get(&before, untouched); err != nil {
		t.Fatal(err)
	}
	sql, err := files.ReadFile("0028_member_trip_list.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = db.Exec(string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err = db.Get(&count, `SELECT count(*) FROM st_group_resource gr JOIN et_resource r ON r.id=gr.resource_id JOIN et_access_level l ON l.id=gr.access_level_id JOIN et_access_scope s ON s.id=gr.access_scope_id WHERE gr.group_id=1 AND r.name='tripsReport' AND l.name='GET' AND s.name='any'`); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if err = db.Get(&count, `SELECT count(*) FROM st_group_resource gr JOIN et_resource r ON r.id=gr.resource_id WHERE gr.group_id=1 AND r.name='tripsReport'`); err != nil || count != 1 {
		t.Fatal("unexpected additional report permissions", count, err)
	}
	var after string
	if err = db.Get(&after, untouched); err != nil || before != after {
		t.Fatal("unrelated permissions changed", err)
	}
}
