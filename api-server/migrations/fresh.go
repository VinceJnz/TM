package migrations

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

//go:embed fresh/baseline.sql
var freshBaseline string

const freshBaselineThrough = "0039_security_admin_lockout.sql"

// ApplyFresh installs the consolidated snapshot for an empty schema only.
// Its ledger entries preserve the upgrade path shared with existing databases.
func ApplyFresh(ctx context.Context, db *sqlx.DB) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(7462026)"); err != nil {
		return err
	}
	var occupied bool
	if err := tx.GetContext(ctx, &occupied, `SELECT EXISTS (
		SELECT 1 FROM pg_class WHERE relnamespace = current_schema()::regnamespace
		UNION ALL
		SELECT 1 FROM pg_proc WHERE pronamespace = current_schema()::regnamespace
	)`); err != nil {
		return err
	}
	if occupied {
		return errors.New("the fresh baseline requires an empty schema")
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE schema_migrations (
		version text PRIMARY KEY, checksum text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, freshBaseline); err != nil {
		return fmt.Errorf("fresh baseline failed: %w", err)
	}
	for _, name := range Names() {
		data, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		checksum := sha256.Sum256(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,checksum,compatible_since)
			VALUES($1,$2,NULLIF($3,''))`, name, hex.EncodeToString(checksum[:]), compatibleSince[name]); err != nil {
			return err
		}
		if name == freshBaselineThrough {
			if err := tx.Commit(); err != nil {
				return err
			}
			return Apply(ctx, db, false)
		}
	}
	return fmt.Errorf("fresh baseline version %s is missing from the migration chain", freshBaselineThrough)
}
