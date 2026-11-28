# Fresh PostgreSQL installation

1. Provision an empty database, a migration owner, and a separate API runtime login. Follow the role and connection settings in [Runtime database permissions](runtime-database-permissions.md).
2. Run the candidate release's `tm-migrate -grant-role <runtime-role>` with the migration owner's credentials. Use the runtime credentials to run `tm-migrate -check-runtime`.
3. From the repository root, connect to the new database as its migration owner with `psql` and apply the initial Admin and SysAdmin grants.

   ```powershell
   psql -X -f scripts/seed-install-grants.sql
   ```

4. Start the same API release and check readiness. Register the designated administrator using the normal signup flow and verify the email address. The account should be awaiting activation.
5. Connect as the database's migration owner with `psql`. Pass the exact verified account username and use the same connection settings as the migration command.

   ```powershell
   psql -X -v username=installation-admin -f scripts/bootstrap-sysadmin.sql
   ```

   The script activates the verified account and adds it to the group identified by the `sysadmin` system key. It refuses to run while an active SysAdmin member exists. Signup and sign-in use the configured authentication provider.
6. Sign in as the new administrator and check **System Administrator → User Group**. Then test Admin-only and member-only accounts against the expected capabilities, including the trip editor and Booking Status.

If no active SysAdmin member remains, the installation owner can run the same script for a newly verified account. If a SysAdmin member remains active but effective permissions were lost, restore the required grants from the installation's approved grant policy in a reviewed migration-owner transaction. Verify sign-in and the security management actions afterward.
