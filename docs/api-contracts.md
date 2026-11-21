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

This is an incremental contract boundary for finance. Other existing views keep
their current DTOs until their features are changed; add them to the generator
when migrating them rather than maintaining another handwritten schema.
