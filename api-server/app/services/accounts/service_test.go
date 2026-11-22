package accounts

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/accounts"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAccountChangesAreAuditedRevokeTokensAndPreserveHistory(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	input := models.AccountWrite{Name: "Synthetic", Username: "synthetic", Email: "old@example.test", AccountStatusID: 2}
	created, err := s.Save(ctx, 10, 0, input, "")
	if err != nil {
		t.Fatal(err)
	}
	input.MemberCode = created.MemberCode
	if _, err = db.Exec(`UPDATE st_users SET user_password='synthetic-hash',provider='google',provider_id='synthetic-provider' WHERE id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO st_token(user_id,name,token) VALUES($1,'session','synthetic-session')`, created.ID); err != nil {
		t.Fatal(err)
	}
	input.Email = "new@example.test"
	currentVersion, _ := s.Store.Get(ctx, created.ID)
	input.Version = currentVersion.Version
	if _, err = s.Save(ctx, 10, created.ID, input, ""); !errors.Is(err, ErrReason) {
		t.Fatal("reason bypass", err)
	}
	input.Username = "invalid-username-that-is-far-too-long"
	if _, err = s.Save(ctx, 10, created.ID, input, "recovery"); !errors.Is(err, ErrValidation) {
		t.Fatal("validation bypass", err)
	}
	input.Username = "synthetic"
	if _, err = s.Save(ctx, 10, created.ID, input, "verified administrator recovery"); err != nil {
		t.Fatal(err)
	}
	var count int
	var hash, provider string
	if err = db.Get(&count, "SELECT count(*) FROM st_token WHERE user_id=$1", created.ID); err != nil || count != 0 {
		t.Fatal("sessions retained", err)
	}
	if err = db.QueryRow("SELECT user_password,provider_id FROM st_users WHERE id=$1", created.ID).Scan(&hash, &provider); err != nil || hash != "synthetic-hash" || provider != "synthetic-provider" {
		t.Fatal("credentials overwritten", err)
	}
	dto, err := s.Store.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(dto)
	if strings.Contains(string(body), "password") || strings.Contains(string(body), "provider") {
		t.Fatal("secret DTO field")
	}
	var audit string
	if err = db.Get(&audit, `SELECT after_data::text FROM at_security_audit WHERE resource='account' AND record_id=$1 AND actor='user:10' AND operation='UPDATE'`, created.ID); err != nil || !strings.Contains(audit, "recovery") || strings.Contains(audit, "password") {
		t.Fatal("unsafe or missing audit", err)
	}
	if _, err = db.Exec("INSERT INTO st_user_group(user_id,group_id) VALUES($1,1)", created.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, 10, created.ID, "remove"); !errors.Is(err, store.ErrReferenced) {
		t.Fatal("history removed", err)
	}
	input.Email = "another@example.test"
	input.MemberCode = ""
	input.Username = "another"
	other, err := s.Save(ctx, 10, 0, input, "")
	if err != nil {
		t.Fatal(err)
	}
	input.Email = "new@example.test"
	input.Version = other.Version
	if _, err = s.Save(ctx, 10, other.ID, input, "recovery"); err == nil {
		t.Fatal("duplicate accepted")
	}
	current, _ := s.Store.Get(ctx, other.ID)
	if current.Email != "another@example.test" {
		t.Fatal("failed transaction changed account")
	}
}
