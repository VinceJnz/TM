package handlerAuth

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	authStore "api-server/v2/modelMethods/authentication"
	"context"
	"strings"
	"testing"
)

func TestStandardMemberMenuIncludesTripList(t *testing.T) {
	db := testdb.Open(t, "")
	if err := migrations.Apply(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email,user_account_status_id) VALUES(100,'Synthetic','synthetic-menu','menu@example.test',2);
 INSERT INTO st_user_group(user_id,group_id) SELECT 100,id FROM st_group WHERE name='User'`); err != nil {
		t.Fatal(err)
	}
	rows, err := (authStore.Store{DB: db}).MenuCapabilities(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range supportedMenuItems(rows) {
		if strings.EqualFold(item.Name, "tripsReport") && strings.EqualFold(item.AccessLevel, "GET") && item.AccessScope == "any" {
			return
		}
	}
	t.Fatal("standard member Trip List capability missing", rows)
}
