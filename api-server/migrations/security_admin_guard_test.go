package migrations

import (
	"context"
	"errors"
	"testing"

	"api-server/v2/internal/testdb"

	"github.com/lib/pq"
)

func TestSecurityAdminGuard(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t, "")
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO st_users(id,name,username,email,user_account_status_id) VALUES
			(101,'First','first','first@example.test',2),
			(102,'Second','second','second@example.test',2);
		INSERT INTO st_user_group(user_id,group_id)
		SELECT u.id,g.id FROM st_users u CROSS JOIN st_group g
		WHERE u.id IN (101,102) AND g.system_key='sysadmin'
	`); err != nil {
		t.Fatal(err)
	}

	first, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback()
	second, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback()
	if _, err := first.Exec(`UPDATE st_users SET user_account_status_id=3 WHERE id=101`); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Exec(`DELETE FROM st_user_group WHERE user_id=102`); err != nil {
		t.Fatal(err)
	}
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	requireLockoutConflict(t, second.Commit())
	_, err = db.Exec(`
		DELETE FROM st_group_resource g USING st_group sg,et_resource r
		WHERE g.group_id=sg.id AND sg.system_key='sysadmin'
		  AND g.resource_id=r.id AND lower(r.name)='securityusergroup'
	`)
	requireLockoutConflict(t, err)
}

func requireLockoutConflict(t *testing.T, err error) {
	t.Helper()
	var databaseError *pq.Error
	if !errors.As(err, &databaseError) || databaseError.Constraint != "effective_security_admin_required" {
		t.Fatalf("expected lockout conflict, got %v", err)
	}
}
