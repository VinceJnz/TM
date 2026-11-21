package profiles

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/profiles"
	"api-server/v2/models"
	"context"
	"errors"
	"strings"
	"testing"
)

type testMailer struct {
	codes map[string]string
	fail  bool
}

func (m *testMailer) SendMail(to, subject, body string) (bool, error) {
	if m.fail {
		return false, errors.New("synthetic failure")
	}
	parts := strings.Split(body, "\n")
	m.codes[to] = parts[len(parts)-1]
	return true, nil
}
func TestProfileEmailChangesRequireBothProofsAndRevokeSessions(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email,user_birth_date) VALUES(1,'Synthetic','synthetic','old@example.test','1990-01-01'),(2,'Other','other','other@example.test','1990-01-01');INSERT INTO st_token(user_id,name,token) VALUES(1,'session','synthetic-session'),(2,'session','other-session')`); err != nil {
		t.Fatal(err)
	}
	m := &testMailer{codes: map[string]string{}}
	s := Service{Store: store.Store{DB: db}, Mailer: m}
	input := models.ProfileWrite{Name: "New name", Email: "new@example.test", BirthDate: "1990-01-01"}
	if err := s.Save(ctx, 1, input); !errors.Is(err, ErrEmailChange) {
		t.Fatal("email bypassed verification", err)
	}
	input.Email = "old@example.test"
	if err := s.Save(ctx, 1, input); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestEmail(ctx, 1, "new@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestEmail(ctx, 1, "new@example.test"); !errors.Is(err, ErrBusy) {
		t.Fatal("resend was unbounded", err)
	}
	current, _ := s.Store.Get(ctx, 1)
	if current.Email != "old@example.test" {
		t.Fatal("unverified email active")
	}
	proof := models.EmailChangeProof{OldCode: m.codes["old@example.test"], NewCode: m.codes["new@example.test"]}
	if proof.OldCode == "" || proof.NewCode == "" {
		t.Fatal("proof delivery absent")
	}
	var sizes int
	if err := db.Get(&sizes, "SELECT octet_length(old_digest)+octet_length(new_digest) FROM st_email_changes WHERE user_id=1"); err != nil || sizes != 64 {
		t.Fatal("proofs not hashed", sizes, err)
	}
	if err := s.ConfirmEmail(ctx, 2, proof); !errors.Is(err, ErrProof) {
		t.Fatal("another user accepted proofs", err)
	}
	if err := s.ConfirmEmail(ctx, 1, models.EmailChangeProof{OldCode: proof.OldCode, NewCode: "wrong"}); !errors.Is(err, ErrProof) {
		t.Fatal("one proof accepted", err)
	}
	if err := s.ConfirmEmail(ctx, 1, proof); err != nil {
		t.Fatal(err)
	}
	current, _ = s.Store.Get(ctx, 1)
	if current.Email != "new@example.test" {
		t.Fatal(current.Email)
	}
	var audit string
	if err := db.Get(&audit, `SELECT actor||':'||before_data::text||':'||after_data::text FROM at_security_audit WHERE resource='profile_email' AND record_id=1`); err != nil || !strings.HasPrefix(audit, "user:1:") || !strings.Contains(audit, "old@example.test") || strings.Contains(audit, "password") {
		t.Fatal("email audit absent or unsafe", err)
	}
	if err := s.ConfirmEmail(ctx, 1, proof); !errors.Is(err, ErrProof) {
		t.Fatal("proof replay accepted", err)
	}
	var count int
	db.Get(&count, "SELECT count(*) FROM st_token WHERE user_id=1")
	if count != 0 {
		t.Fatal("sessions retained")
	}
	db.Get(&count, "SELECT count(*) FROM st_token WHERE user_id=2")
	if count != 1 {
		t.Fatal("another user's sessions revoked")
	}
	m.fail = true
	if err := s.RequestEmail(ctx, 1, "next@example.test"); !errors.Is(err, ErrMail) {
		t.Fatal(err)
	}
	db.Get(&count, "SELECT count(*) FROM st_email_changes WHERE user_id=1")
	if count != 0 {
		t.Fatal("failed delivery left usable verification")
	}
}
func TestEmailValidationRejectsHeaderInjection(t *testing.T) {
	for _, value := range []string{"", "a@example.test\r\nBcc: victim@example.test", "Name <a@example.test>", "a@example.test,b@example.test"} {
		if _, err := CanonicalEmail(value); err == nil {
			t.Fatal(value)
		}
	}
}

func TestExpiredExhaustedAndConflictingChangesPreserveAccount(t *testing.T) {
	for _, scenario := range []string{"expired", "exhausted", "duplicate"} {
		t.Run(scenario, func(t *testing.T) {
			db := testdb.Open(t, "")
			ctx := context.Background()
			if err := migrations.Apply(ctx, db, false); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'One','one','old@example.test'),(2,'Two','two','taken@example.test'); INSERT INTO st_token(user_id,name,token) VALUES(1,'session','preserve-me')`); err != nil {
				t.Fatal(err)
			}
			m := &testMailer{codes: map[string]string{}}
			s := Service{Store: store.Store{DB: db}, Mailer: m}
			target := "new@example.test"
			if scenario == "duplicate" {
				target = "taken@example.test"
			}
			if err := s.RequestEmail(ctx, 1, target); err != nil {
				t.Fatal(err)
			}
			proof := models.EmailChangeProof{OldCode: m.codes["old@example.test"], NewCode: m.codes[target]}
			if scenario == "expired" {
				if _, err := db.Exec("UPDATE st_email_changes SET expires_at=CURRENT_TIMESTAMP-interval '1 second'"); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "exhausted" {
				for i := 0; i < 5; i++ {
					if err := s.ConfirmEmail(ctx, 1, models.EmailChangeProof{}); !errors.Is(err, ErrProof) {
						t.Fatal(err)
					}
				}
			}
			if err := s.ConfirmEmail(ctx, 1, proof); err == nil {
				t.Fatal("invalid transition accepted")
			}
			current, err := s.Store.Get(ctx, 1)
			if err != nil || current.Email != "old@example.test" {
				t.Fatal(current, err)
			}
			var count int
			if err := db.Get(&count, "SELECT count(*) FROM st_token WHERE user_id=1"); err != nil || count != 1 {
				t.Fatal("session invalidated by failed transition", err)
			}
		})
	}
}
