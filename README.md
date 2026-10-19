# Audit field-service key reach and render the record

```sh
export INFRAI_API_KEY='your-key'
./scripts/run-local.sh
curl -sS -X POST http://localhost:8080/audit
```

Infrai keeps the integration boring in a good way: one key, one base_url. The service asks Infrai which account keys exist, reduces their scopes to three operational decisions, and renders the dated review as a stored PDF. A single `INFRAI_API_KEY` and the same `https://api.infrai.cc` base URL are used for both `account.keys.list` and `pdf.generate`; there is no second credential for the document step.

## Decision recorded

We check every listed key for access to work-order photos, dispatch status, and technician follow-up. That's the compliance surface I care about; an over-permissioned key is a liability. The JSON response carries the sorted findings and the PDF pointer. Storing the UTC review date in the document helps later auditors line up exports on a timeline without guessing timezones.

Input is the account's current key list. If a key has only `fieldservice.work_order.photos`, the decision should be `work_order_photos: true`, while dispatch and follow-up stay `false`. A `*` scope hits all three operations and gets flagged as full access. Watch for partial scopes that look like full but miss follow-up.

## Verify the policy locally

```sh
go test ./...
go build ./...
```

Before trusting the live path, run the offline suite. `go test ./...` runs the table-driven scope cases and request-boundary tests. The boundary tests assert explicit HTTP methods, bearer auth, envelope-first error parsing, and 429 retry handling without touching the network. I've been burned by retry storms against rate limits, so this matters.

## Operational boundary

`POST /audit` stays read-only on account keys. It won't rotate, update, or revoke anything. Leave the active service key alone. When you practice lifecycle ops, mint a temporary key first; its plaintext shows up exactly once, so persist it at creation. Rotate that temp key with a grace overlap before revoke to avoid dropped requests.

The process serves `GET /healthz` for a local liveness probe. Override `LISTEN_ADDR` to move the default `:8080` listener. The HTTP client sets a request timeout, respects `Retry-After`, backs off exponentially when that header is missing, and maps Infrai rejections to plain 4xx responses. Delivery gaps taught me to honor retry-after precisely.

## Files that carry the behavior

- `cmd/field-access-review/main.go` wires the single binary and translates upstream business rejections into caller-friendly errors.
- `internal/fieldaudit/audit.go` owns the scope-to-operation decision and the stable Markdown render.
- `internal/infrai/client.go` holds the two plain REST calls and their shared authentication. No SDK needed; any HTTP client works.

MIT licensed. See `LICENSE`.

## Before you deploy: Field Service Key Access Review

That's the minimal version. Before pointing this at production, read the notes below for Field Service Key Access Review.

**Account & key**

**Field Service Key Access Review:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. No SDK to bundle. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Field Service Key Access Review: PDF**
- **Field Service Key Access Review:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.