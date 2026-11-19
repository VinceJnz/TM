package migrations

import (
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// GrantRuntimeAccess is an explicit operator action, not application startup.
// Migration history stays read-only to the API role.
func GrantRuntimeAccess(ctx context.Context, db *sqlx.DB, role string) error {
	var privileged bool
	if err := db.GetContext(ctx, &privileged, "SELECT rolsuper OR rolcreaterole OR rolcreatedb FROM pg_roles WHERE rolname=$1", role); err != nil {
		return errors.New("runtime role does not exist")
	}
	if privileged {
		return errors.New("runtime role must not have administrative privileges")
	}
	var schema string
	if err := db.GetContext(ctx, &schema, "SELECT current_schema()"); err != nil {
		return err
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	name := pq.QuoteIdentifier(role)
	if _, err := tx.ExecContext(ctx, "GRANT USAGE ON SCHEMA "+pq.QuoteIdentifier(schema)+" TO "+name); err != nil {
		return err
	}
	var tables []string
	if err := tx.SelectContext(ctx, &tables, `SELECT table_name FROM information_schema.tables WHERE table_schema=current_schema() AND left(table_name,3) IN('at_','et_','st_')`); err != nil {
		return err
	}
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, "GRANT SELECT,INSERT,UPDATE,DELETE ON "+pq.QuoteIdentifier(table)+" TO "+name); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "GRANT SELECT ON schema_migrations TO "+name); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA "+pq.QuoteIdentifier(schema)+" TO "+name); err != nil {
		return err
	}
	return tx.Commit()
}
