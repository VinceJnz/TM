# API Coding Standards

This document is the working contract for the Go API. It is deliberately incremental: existing endpoints remain compatible until they are migrated, while all new or materially changed API code follows these rules.

## Package responsibilities

| Area | Responsibility | Must not contain |
| --- | --- | --- |
| `main.go` | composition, middleware, server lifecycle, route registration | business or SQL logic |
| `localHandlers/handler<Domain>` | HTTP translation: decode, validate, call application/data code, respond | duplicated response formatting or connection setup |
| `localHandlers/helpers` | HTTP-wide helpers: response, decoding, middleware, routing | domain-specific rules |
| `modelMethods` | database access and transaction boundaries | `http.ResponseWriter`, request decoding, or HTTP status selection |
| `models` | request/response and persistence data types | database queries or HTTP behaviour |
| `app/gateways` | integrations with external systems | handler-specific logic |

The current `dbStandardTemplate` is legacy HTTP-aware shared CRUD code. Do not extend that pattern in new domains. New work should keep database functions HTTP-independent and let handlers map errors to HTTP responses.

## HTTP contract

Success responses return the resource body directly. Errors use one JSON shape:

```json
{"error":{"code":"invalid_request","message":"Invalid request payload"}}
```

Use `helpers.WriteJSON` and `helpers.WriteError`; never call `WriteHeader` before setting headers, and never write a second response. Error `code` is stable and machine-readable (`invalid_request`, `validation_failed`, `not_found`, `conflict`, `unauthorized`, `forbidden`, `internal_error`); `message` is safe for clients. Log the underlying error on the server, not in the response.

## Handler shape

Each handler follows this order: obtain and validate route input, authenticate/authorise, decode one request DTO, validate it, call application/data code with `r.Context()`, map the result to one response. Parsing helpers should return `(value, error)` rather than write a response themselves, so execution cannot continue after a failed parse.

Use plural, kebab-case resource paths and standard methods: `GET /resources`, `GET /resources/{id}`, `POST /resources`, `PUT /resources/{id}`, `DELETE /resources/{id}`. Register a domain's routes in one `RegisterRoutes` method.

## Data and errors

Pass `context.Context` into all new database and gateway calls. Use `QueryContext`, `GetContext`, `SelectContext`, `ExecContext`, and `BeginTxx`. Return wrapped errors (`fmt.Errorf("load trip %d: %w", id, err)`) from lower layers, and classify expected errors with sentinel errors or typed errors rather than matching error strings.

Treat an empty collection as `200` with `[]`, not `404`. Use `404` for a requested single resource that does not exist, `409` for conflicts, `422` for valid JSON that fails domain validation, and `500` for unexpected faults.

## Naming, tests, and change discipline

Use package names and directories based on domains, exported Go names only when another package needs them, and short receiver names. Prefer `TripCreateRequest` / `TripResponse` DTOs over exposing persistence structs when their contracts differ. Keep SQL near its repository/data function and use named constants only for stable queries.

Every new shared helper needs focused `httptest` coverage. Every handler change needs success plus relevant invalid-input, unauthenticated/unauthorised, not-found, and conflict tests. Run `gofmt -w` on changed Go files and `go test ./...` from `api-server` before handoff.

## Migration order

1. Use the response helpers for all new endpoints and migrate existing endpoints by domain.
2. Replace `GetID(w, r) int` with an error-returning parser at each migrated handler.
3. Move each domain's SQL from HTTP-aware templates into context-aware data functions.
4. Extract services only where a use case coordinates multiple repositories or gateways.

Do not introduce a framework, a global generic repository, or a service layer for simple one-query endpoints; clarity of the domain boundary is more valuable than abstraction count.
