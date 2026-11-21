package migrations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

//go:embed *.sql
var files embed.FS

const Latest = "0023_account_administration_audit.sql"

func Names() []string {
	entries, _ := files.ReadDir(".")
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

// Apply never guesses a baseline for an existing database. Adoption is explicit,
// validates required API columns and records only the baseline before upgrades.
func Apply(ctx context.Context, db *sqlx.DB, adopt bool) error {
	conn, err := db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock(7462026)"); err != nil {
		return err
	}
	defer func() {
		unlock, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlock, "SELECT pg_advisory_unlock(7462026)"); err != nil {
			conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version text PRIMARY KEY,checksum text NOT NULL,applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	var recorded []string
	if err := conn.SelectContext(ctx, &recorded, "SELECT version FROM schema_migrations"); err != nil {
		return err
	}
	known := map[string]bool{}
	for _, name := range Names() {
		known[name] = true
	}
	for _, name := range recorded {
		if !known[name] {
			return fmt.Errorf("unknown database migration: %s", name)
		}
	}
	for _, name := range Names() {
		data, _ := files.ReadFile(name)
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])
		var previous string
		err := conn.GetContext(ctx, &previous, "SELECT checksum FROM schema_migrations WHERE version=$1", name)
		if err == nil {
			if checksum != previous {
				return fmt.Errorf("migration checksum changed: %s", name)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if name == "0001_baseline.sql" {
			var count int
			if err := conn.GetContext(ctx, &count, `SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name<>'schema_migrations'`); err != nil {
				return err
			}
			if count > 0 {
				if !adopt {
					return errors.New("existing schema has no migration baseline; audit it and explicitly adopt before upgrading")
				}
				for _, table := range []string{"st_users", "st_token", "st_group", "st_group_resource", "st_user_group", "et_resource", "et_access_level", "et_access_scope", "at_bookings", "at_booking_people", "at_trips", "at_payments", "at_refunds"} {
					var exists bool
					if err := conn.GetContext(ctx, &exists, "SELECT to_regclass($1) IS NOT NULL", table); err != nil || !exists {
						return fmt.Errorf("cannot adopt: required table %s missing", table)
					}
				}
				for table, columns := range map[string][]string{"st_users": {"provider", "provider_id", "user_account_status_id"}, "st_token": {"token", "session_data", "valid_from", "valid_to"}, "st_group_resource": {"access_scope_id"}, "at_bookings": {"stripe_session_id", "booking_price", "owner_id"}, "at_payments": {"stripe_session_id", "booking_id", "amount"}} {
					for _, column := range columns {
						var exists bool
						if err := conn.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=$1 AND column_name=$2)`, table, column); err != nil || !exists {
							return fmt.Errorf("cannot adopt: required column %s.%s missing", table, column)
						}
					}
				}
				differences, err := compareCatalog(ctx, conn)
				if err != nil {
					return fmt.Errorf("cannot inspect baseline compatibility: %w", err)
				}
				if len(differences) > 0 {
					return fmt.Errorf("baseline has %d compatibility differences; run tm-migrate -check-existing and reconcile them before adoption", len(differences))
				}
				data = nil // Never recreate an adopted schema.
			}
		}
		tx, err := conn.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		if len(data) > 0 {
			if _, err := tx.ExecContext(ctx, string(data)); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %s failed: %w", name, err)
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", name, checksum); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func Check(ctx context.Context, db *sqlx.DB) error {
	var recorded []struct {
		Version  string `db:"version"`
		Checksum string `db:"checksum"`
	}
	if err := db.SelectContext(ctx, &recorded, "SELECT version,checksum FROM schema_migrations"); err != nil {
		return errors.New("database migrations are required")
	}
	names := Names()
	if len(recorded) != len(names) {
		return errors.New("database migrations are not current")
	}
	checksums := map[string]string{}
	for _, name := range names {
		data, _ := files.ReadFile(name)
		sum := sha256.Sum256(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
		checksums[name] = hex.EncodeToString(sum[:])
	}
	for _, entry := range recorded {
		if checksums[entry.Version] != entry.Checksum {
			return errors.New("database migration checksum mismatch")
		}
	}
	return nil
}
