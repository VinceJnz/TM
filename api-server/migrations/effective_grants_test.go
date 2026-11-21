package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"github.com/lib/pq"
	"testing"
)

func TestRuntimeRejectsUnexpectedEffectiveGrants(t *testing.T) {
	for _, kind := range []string{"public", "inherited", "column", "sequence", "grant-option"} {
		t.Run(kind, func(t *testing.T) {
			db := testdb.Open(t, "")
			ctx := context.Background()
			if err := Apply(ctx, db, false); err != nil {
				t.Fatal(err)
			}
			var schema string
			db.Get(&schema, "SELECT current_schema()")
			role := schema + "_api"
			parent := schema + "_parent"
			if _, err := db.Exec("CREATE ROLE " + pq.QuoteIdentifier(role) + ";CREATE ROLE " + pq.QuoteIdentifier(parent)); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				db.Exec("DROP OWNED BY " + pq.QuoteIdentifier(role) + "," + pq.QuoteIdentifier(parent))
				db.Exec("DROP ROLE " + pq.QuoteIdentifier(role) + "," + pq.QuoteIdentifier(parent))
			})
			query := ""
			switch kind {
			case "public":
				query = "GRANT DELETE ON at_checkout_quotes TO PUBLIC"
			case "inherited":
				query = "GRANT " + pq.QuoteIdentifier(parent) + " TO " + pq.QuoteIdentifier(role) + ";GRANT DELETE ON at_checkout_quotes TO " + pq.QuoteIdentifier(parent)
			case "column":
				query = "GRANT " + pq.QuoteIdentifier(parent) + " TO " + pq.QuoteIdentifier(role) + ";GRANT UPDATE(amount_minor) ON at_checkout_attempts TO " + pq.QuoteIdentifier(parent)
			case "sequence":
				query = "CREATE SEQUENCE private_sequence;GRANT USAGE ON SEQUENCE private_sequence TO PUBLIC"
			case "grant-option":
				query = "GRANT " + pq.QuoteIdentifier(parent) + " TO " + pq.QuoteIdentifier(role) + ";GRANT SELECT ON at_checkout_quotes TO " + pq.QuoteIdentifier(parent) + " WITH GRANT OPTION"
			}
			if _, err := db.Exec(query); err != nil {
				t.Fatal(err)
			}
			if err := GrantRuntimeAccess(ctx, db, role); err == nil {
				t.Fatal("excess privilege accepted")
			}
		})
	}
}
