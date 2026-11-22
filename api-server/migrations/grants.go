package migrations

import (
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"strings"
)

// Explicit capabilities: adding a table never silently grants runtime access.
func runtimePolicy() map[string]string {
	policy := map[string]string{}
	for _, name := range strings.Fields(`at_booking_people at_bookings at_group_bookings at_trip_cost_groups at_trip_costs at_trips at_payments at_refunds at_booking_vouchers at_checkout_reservations et_access_level et_access_scope et_booking_status et_member_status et_resource et_seasons et_trip_difficulty et_trip_status et_trip_type et_user_account_status et_user_age_groups st_group st_group_resource st_token st_user_credentials st_user_group st_users st_auth_rate_limits st_email_changes`) {
		policy[name] = "SELECT,INSERT,UPDATE,DELETE"
	}
	policy["st_account_review_requests"] = "SELECT,UPDATE (dispatched_at)"
	policy["schema_migrations"] = "SELECT"
	policy["at_financial_audit"] = "SELECT"
	policy["at_security_audit"] = "SELECT"
	policy["at_notification_outbox"] = "SELECT,INSERT,UPDATE (attempts,available_at,sent_at)"
	for _, name := range []string{"at_stripe_events", "at_checkout_quotes", "at_checkout_reconciliations", "at_voucher_redemptions"} {
		policy[name] = "SELECT,INSERT"
	}
	policy["at_checkout_attempts"] = "SELECT,INSERT,UPDATE (session_id,checkout_url,state,reconcile_after,reconcile_failures)"
	policy["at_provider_refunds"] = "SELECT,INSERT,UPDATE (status,amount_minor,checked_at,reconcile_after,reconcile_failures)"
	policy["at_payment_reviews"] = "SELECT,INSERT,UPDATE (last_seen,failures,resolved_at,resolution,actor,evidence)"
	return policy
}

// This explicit operator command replaces previous broad grants atomically.
func GrantRuntimeAccess(ctx context.Context, db *sqlx.DB, role string) error {
	var privileged bool
	if err := db.GetContext(ctx, &privileged, `SELECT EXISTS(SELECT 1 FROM pg_roles r WHERE (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb) AND pg_has_role($1::text,r.oid,'MEMBER'))`, role); err != nil {
		return errors.New("runtime role unavailable")
	}
	if privileged {
		return errors.New("runtime role must not inherit administrative privileges")
	}
	var schema string
	if err := db.GetContext(ctx, &schema, `SELECT current_schema()`); err != nil {
		return err
	}
	var owner bool
	if err := db.GetContext(ctx, &owner, `SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$2 AND pg_has_role($1::text,c.relowner,'MEMBER')) OR EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$2 AND pg_has_role($1::text,nspowner,'MEMBER'))`, role, schema); err != nil {
		return err
	}
	if owner {
		return errors.New("runtime role must not own or inherit ownership of application objects")
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	name := pq.QuoteIdentifier(role)
	namespace := pq.QuoteIdentifier(schema)
	if _, err = tx.ExecContext(ctx, "REVOKE CREATE ON SCHEMA "+namespace+" FROM "+name); err != nil {
		return err
	}
	var canCreate bool
	if err = tx.GetContext(ctx, &canCreate, `SELECT has_schema_privilege($1,$2,'CREATE')`, role, schema); err != nil {
		return err
	}
	if canCreate {
		return errors.New("remove inherited/PUBLIC schema CREATE before granting runtime access")
	}
	if _, err = tx.ExecContext(ctx, "GRANT USAGE ON SCHEMA "+namespace+" TO "+name); err != nil {
		return err
	}
	var tables []string
	if err = tx.SelectContext(ctx, &tables, `SELECT table_name FROM information_schema.tables WHERE table_schema=current_schema() AND (left(table_name,3) IN ('at_','et_','st_') OR table_name='schema_migrations')`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA "+namespace+" FROM "+name); err != nil {
		return err
	}
	policy := runtimePolicy()
	expectedSequences := map[string]bool{}
	for _, table := range tables {
		qualified := namespace + "." + pq.QuoteIdentifier(table)
		if _, err = tx.ExecContext(ctx, "REVOKE ALL PRIVILEGES ON "+qualified+" FROM "+name); err != nil {
			return err
		}
		privileges, known := policy[table]
		if !known {
			continue
		}
		if _, err = tx.ExecContext(ctx, "GRANT "+privileges+" ON "+qualified+" TO "+name); err != nil {
			return err
		}
		if strings.Contains(privileges, "INSERT") {
			var sequences []string
			// Legacy dumps can reference nextval() without an OWNED BY link.
			// Include default-expression dependencies as well as serial/identity
			// ownership, without granting unrelated sequences in the schema.
			if err = tx.SelectContext(ctx, &sequences, `SELECT DISTINCT quote_ident(n.nspname)||'.'||quote_ident(s.relname)
FROM pg_class s JOIN pg_namespace n ON n.oid=s.relnamespace
WHERE s.relkind='S' AND (
 EXISTS(SELECT 1 FROM pg_depend d JOIN pg_attrdef a ON d.classid='pg_attrdef'::regclass AND d.objid=a.oid
        WHERE a.adrelid=$1::text::regclass AND d.refclassid='pg_class'::regclass AND d.refobjid=s.oid)
 OR EXISTS(SELECT 1 FROM pg_depend d WHERE d.classid='pg_class'::regclass AND d.objid=s.oid
        AND d.refclassid='pg_class'::regclass AND d.refobjid=$1::text::regclass AND d.deptype IN ('a','i'))
)`, qualified); err != nil {
				return err
			}
			for _, sequence := range sequences {
				expectedSequences[sequence] = true
				if _, err = tx.ExecContext(ctx, "GRANT USAGE,SELECT ON SEQUENCE "+sequence+" TO "+name); err != nil {
					return err
				}
			}
		}
	}
	if _, err = tx.ExecContext(ctx, "GRANT EXECUTE ON FUNCTION "+namespace+".redact_notification_payloads(integer,integer,integer) TO "+name); err != nil {
		return err
	}
	if err = verifyRuntimePrivileges(ctx, tx, role, tables, expectedSequences); err != nil {
		return err
	}
	return tx.Commit()
}
