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

This is an incremental contract boundary for finance. Other existing views keep
their current DTOs until their features are changed; add them to the generator
when migrating them rather than maintaining another handwritten schema.
