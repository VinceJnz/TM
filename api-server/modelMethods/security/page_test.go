package security

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"api-server/v2/modelMethods/resourcequery"
	"context"
	"testing"
)

func TestParentFilterPrecedesPagination(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) SELECT n,'Synthetic','synthetic'||n,'person'||n||'@example.test' FROM generate_series(1,130)n; INSERT INTO st_user_group(user_id,group_id) SELECT n,1 FROM generate_series(1,130)n; INSERT INTO st_user_group(user_id,group_id) VALUES(130,2)`); err != nil {
		t.Fatal(err)
	}
	s := Store{DB: db}
	rows, info, err := s.Members(ctx, resourcequery.Page{Limit: 1}, 130)
	if err != nil || len(rows) != 1 || rows[0].UserID != 130 || info.NextOffset != 1 {
		t.Fatal(rows, info, err)
	}
	next, _, err := s.Members(ctx, resourcequery.Page{Limit: 1, Offset: 1}, 130)
	if err != nil || len(next) != 1 || next[0].UserID != 130 || next[0].ID == rows[0].ID {
		t.Fatal(next, err)
	}
	cursor, _, err := s.Members(ctx, resourcequery.Page{Limit: 1, Cursor: true, After: int64(rows[0].ID)}, 130)
	if err != nil || len(cursor) != 1 || cursor[0].ID != next[0].ID {
		t.Fatal(cursor, err)
	}
	grants, _, err := s.Grants(ctx, resourcequery.Page{Limit: 500}, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range grants {
		if g.GroupID != 2 {
			t.Fatal("cross-parent grant")
		}
	}
}
