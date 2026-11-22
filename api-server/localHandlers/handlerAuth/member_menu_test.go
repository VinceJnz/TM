package handlerAuth

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"api-server/v2/models"
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
	var rows []models.MenuItem
	if err := db.Select(&rows, sqlMenuList, 100, models.AccountActive); err != nil {
		t.Fatal(err)
	}
	for _, item := range supportedMenuItems(rows) {
		if strings.EqualFold(item.Name, "tripsReport") && strings.EqualFold(item.AccessLevel, "GET") && item.AccessScope == "any" {
			return
		}
	}
	t.Fatal("standard member Trip List capability missing", rows)
}
