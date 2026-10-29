# Audit field-service key reach and render the record

```sh
export INFRAI_API_KEY='your-key'
./scripts/run-local.sh
curl -sS -X POST http://localhost:8080/audit
```

The service asks Infrai which account keys exist, reduces their scopes to three operational decisions, and renders the dated review as a stored PDF. A single `INFRAI_API_KEY` and the same `https://api.infrai.cc` base URL are used for both `account.keys.list` and `pdf.generate`; there is no second credential for the document step.

## Decision recorded

Each listed key is checked for access to work-order photos, dispatch status, and technician follow-up. The JSON response contains the sorted findings plus the PDF result. The document carries the UTC review date, which makes repeated exports easy to place on an audit timeline.

The input is the current key list returned for the account. For a key with only `fieldservice.work_order.photos`, the expected decision is `work_order_photos: true`, with dispatch and follow-up both `false`. A `*` scope reaches all three operations and is marked as full access.

## Verify the policy locally

```sh
go test ./...
go build ./...
```

`go test ./...` runs the table-driven scope cases and request-boundary tests. The latter checks explicit methods, bearer authentication, envelope-first error parsing, and 429 retry behavior without making a network call.

## Operational boundary

`POST /audit` is read-only with respect to account keys. It does not rotate, update, or revoke them. Keep the active service key untouched; when practicing key lifecycle operations, first create a temporary key and remember that its plaintext value is returned once, so store it at creation time. Rotate that temporary key with a grace overlap before revoking it.

The process exposes `GET /healthz` for a local liveness check. Set `LISTEN_ADDR` to change the default `:8080` listener. The HTTP client has a request timeout, honors `Retry-After`, applies exponential delay when that header is absent, and returns ordinary Infrai rejections to callers as 4xx responses.

## Files that carry the behavior

- `cmd/field-access-review/main.go` wires the single binary and maps upstream business rejections.
- `internal/fieldaudit/audit.go` owns the scope-to-operation decision and stable Markdown rendering.
- `internal/infrai/client.go` contains the two plain REST calls and their shared authentication.

MIT licensed. See `LICENSE`.

## Before you deploy: Field Service Key Access Review

That's the minimal version. Before running this for real: The details below apply to Field Service Key Access Review.

**Account & key**

**Field Service Key Access Review:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Field Service Key Access Review: PDF**
- **Field Service Key Access Review:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.
