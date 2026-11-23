# API contracts

The payment/refund path uses server write DTOs as its source of mutable fields.
From api-server, run `go run ./cmd/contracts` after an intentional contract
change. Commit the generated frontend DTOs and `contracts/finance.openapi.json`.
Use `go run ./cmd/contracts -check` to detect drift without modifying files.
Both local release verification and CI enforce this check.

The OpenAPI 3.1 document describes finance paths relative to the runtime API
prefix, session authentication, read arrays, write fields, and provider-record
conflicts. Browser finance views share generated DTOs and an API adapter that
propagates failed writes. Money is authoritative and exact on the server;
browser numeric fields remain display/input values for compatibility.

Payment and refund handlers delegate reads to SQL-only `modelMethods/payments`
and `modelMethods/refunds` stores. The handlers own HTTP validation, permission
checks and pagination headers; stores receive request contexts and return data
or errors. Financial writes remain transactional commands in `paymentLedger`,
with provider orchestration in the checkout service. Shared route-ID parsing
does not perform database access. This separation preserves existing routes,
JSON fields, array responses, and offset/cursor navigation.

Generated contracts now also cover bookings, My Bookings, trips and the Trip List
participant report. Those views share generated read DTOs; booking/trip write
projections include only fields accepted by explicit server write DTOs. Decimal
JSON strings and nullable fields keep their existing wire representations. The
legacy finance-named output files now contain these resource contracts as well.
The schema describes pagination, nested trip-booking reads, report reads and
required finance correction headers. Extend this generator when migrating other
resources rather than maintaining another handwritten schema.

Pricing now follows the same pattern: TripCostDTO, TripCostGroupDTO and explicit
write contracts are generated from server types. Both pricing views use those
types, excluding IDs, timestamps and joined display fields from writes. Server
amounts use exact decimals while preserving the existing JSON number contract.
Service/handler regressions validate rejected amounts, references, any-scope
access and deletion conflicts; client tests verify nullable display fields and
the actual mutable-field projection. Legacy resources outside this migration
remain outside the generated contract coverage.

Security grant and membership views now also share generated read/write DTOs.
Their schemas specify required positive IDs, supported scopes and permission
resources. Pricing metadata includes description lengths and currency bounds.
Generated operations include validation/conflict/dependency responses; legacy
plain-text errors remain represented alongside structured errors. Behavioral
tests compare security handler validation and read-array contracts with generated
metadata. The generator does not replace database/reference or multi-step domain
validation. Extend metadata and behavioral tests with each resource migration.

For voucher and group booking edits, PUT requires the version from the most
recent read. A missing or stale version returns HTTP 409 with error code
stale_record. Creation ignores the version and the database starts it at 1.
Existing response shapes are preserved with an additional version field.
