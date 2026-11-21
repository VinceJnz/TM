package security

import (
	"api-server/v2/internal/observability"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/security"
	"api-server/v2/models"
	"context"
	"errors"
	"testing"
)

func TestSecurityChangesValidateAndAuditAtomically(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := observability.JobContext(context.Background())
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(100,'Synthetic','synthetic','synthetic@example.test')`); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	v, err := s.SaveMember(ctx, 100, 0, models.UserGroupWrite{UserID: 100, GroupID: 1})
	if err != nil {
		t.Fatal(err)
	}
	var actor, id string
	if err = db.QueryRow(`SELECT actor,request_id FROM at_security_audit WHERE resource='st_user_group' AND record_id=$1`, v.ID).Scan(&actor, &id); err != nil || actor != "user:100" || id != observability.RequestID(ctx) {
		t.Fatal(actor, id, err)
	}
	if _, err = s.SaveMember(ctx, 100, v.ID, models.UserGroupWrite{UserID: 99999, GroupID: 1}); err == nil {
		t.Fatal("invalid reference accepted")
	}
	if err = s.Delete(ctx, 100, v.ID, false); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM at_security_audit WHERE resource='st_user_group' AND record_id=$1`, v.ID)
	if count != 2 {
		t.Fatal("failed mutation audited or audit lost", count)
	}
	var resource, method, scope int
	db.Get(&resource, "SELECT id FROM et_resource WHERE name='securityUserGroup'")
	db.Get(&method, "SELECT id FROM et_access_level WHERE name='POST'")
	db.Get(&scope, "SELECT id FROM et_access_scope WHERE name='own'")
	if _, err = s.SaveGrant(ctx, 100, 0, models.GroupResourceWrite{GroupID: 1, ResourceID: resource, AccessLevelID: method, AccessScopeID: scope}); !errors.Is(err, ErrValidation) {
		t.Fatal("unsupported own grant accepted", err)
	}
	if _, err = db.Exec("DELETE FROM at_security_audit"); err == nil {
		t.Fatal("audit rewritten")
	}
}

func TestGroupDefinitionsUseValidatedAuditedTransactions(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := observability.JobContext(context.Background())
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	if _, err := s.SaveGroup(ctx, 100, 0, models.GroupWrite{Name: " "}); !errors.Is(err, ErrValidation) {
		t.Fatal(err)
	}
	v, err := s.SaveGroup(ctx, 100, 0, models.GroupWrite{Name: " Synthetic ", Description: "Testing"})
	if err != nil || v.Name != "Synthetic" {
		t.Fatal(v, err)
	}
	if _, err = db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(100,'Synthetic','synthetic','synthetic@example.test')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO st_user_group(user_id,group_id) VALUES(100,$1)`, v.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteGroup(ctx, 100, v.ID); err == nil {
		t.Fatal("referenced group was deleted")
	}
	if _, err = s.SaveGroup(ctx, 100, v.ID, models.GroupWrite{Name: "Updated"}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.Get(&count, `SELECT count(*) FROM at_security_audit WHERE resource='st_group' AND record_id=$1 AND actor='user:100'`, v.ID); err != nil || count != 2 {
		t.Fatal("group audit missing", count, err)
	}
	if _, err = db.Exec("DELETE FROM st_user_group WHERE group_id=$1", v.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteGroup(ctx, 100, v.ID); err != nil {
		t.Fatal(err)
	}
}
