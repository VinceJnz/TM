package migrations

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"strings"
)

// Check effective access, including inherited and PUBLIC privileges. Do not
// silently revoke permissions belonging to other roles or change memberships.
func verifyRuntimePrivileges(ctx context.Context, tx *sqlx.Tx, role string, tables []string, sequences map[string]bool) error {
	var schema string
	if err := tx.GetContext(ctx, &schema, "SELECT current_schema()"); err != nil {
		return err
	}
	for _, table := range tables {
		object := pq.QuoteIdentifier(schema) + "." + pq.QuoteIdentifier(table)
		policy := runtimePolicy()[table]
		full := policy
		if i := strings.Index(full, "UPDATE ("); i >= 0 {
			full = full[:i]
		}
		for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"} {
			allowed := false
			for _, entry := range strings.Split(full, ",") {
				if entry == priv {
					allowed = true
				}
			}
			var actual, grant bool
			if err := tx.GetContext(ctx, &actual, "SELECT has_table_privilege($1,$2,$3)", role, object, priv); err != nil {
				return err
			}
			if err := tx.GetContext(ctx, &grant, "SELECT has_table_privilege($1,$2,$3)", role, object, priv+" WITH GRANT OPTION"); err != nil {
				return err
			}
			if actual != allowed || grant {
				return fmt.Errorf("unexpected effective %s privilege on %s; review inherited/PUBLIC grants", priv, table)
			}
		}
		var columns []string
		if err := tx.SelectContext(ctx, &columns, "SELECT attname FROM pg_attribute WHERE attrelid=$1::text::regclass AND attnum>0 AND NOT attisdropped", object); err != nil {
			return err
		}
		for _, column := range columns {
			for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "REFERENCES"} {
				allowed := false
				for _, entry := range strings.Split(full, ",") {
					if entry == priv {
						allowed = true
					}
				}
				if priv == "UPDATE" && strings.Contains(policy, "UPDATE (") {
					list := strings.Split(strings.TrimSuffix(strings.SplitN(policy, "UPDATE (", 2)[1], ")"), ",")
					for _, entry := range list {
						if entry == column {
							allowed = true
						}
					}
				}
				var actual, grant bool
				if err := tx.GetContext(ctx, &actual, "SELECT has_column_privilege($1,$2,$3,$4)", role, object, column, priv); err != nil {
					return err
				}
				if err := tx.GetContext(ctx, &grant, "SELECT has_column_privilege($1,$2,$3,$4)", role, object, column, priv+" WITH GRANT OPTION"); err != nil {
					return err
				}
				if actual != allowed || grant {
					return fmt.Errorf("unexpected effective %s privilege on %s.%s; review column/inherited/PUBLIC grants", priv, table, column)
				}
			}
		}
	}
	var all []string
	if err := tx.SelectContext(ctx, &all, `SELECT quote_ident(n.nspname)||'.'||quote_ident(c.relname) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='S' AND n.nspname=current_schema()`); err != nil {
		return err
	}
	// Referenced legacy sequences may live outside the application schema.
	for sequence := range sequences {
		found := false
		for _, name := range all {
			if name == sequence {
				found = true
			}
		}
		if !found {
			all = append(all, sequence)
		}
	}
	for _, sequence := range all {
		for _, priv := range []string{"USAGE", "SELECT", "UPDATE"} {
			var actual, grant bool
			if err := tx.GetContext(ctx, &actual, "SELECT has_sequence_privilege($1,$2,$3)", role, sequence, priv); err != nil {
				return err
			}
			if err := tx.GetContext(ctx, &grant, "SELECT has_sequence_privilege($1,$2,$3)", role, sequence, priv+" WITH GRANT OPTION"); err != nil {
				return err
			}
			if actual != (sequences[sequence] && priv != "UPDATE") || grant {
				return fmt.Errorf("unexpected effective sequence %s privilege on %s", priv, sequence)
			}
		}
	}
	return nil
}
