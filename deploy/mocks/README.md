# Mocks — Prism + Meta Graph API

Self-contained mock server for the Meta Graph API (WhatsApp Business Messaging, Flows, Commerce, etc.), driven by the official spec at `documentos/business-messaging-api_v23.0.yaml`. We use [Stoplight Prism](https://github.com/stoplightio/prism) so we can validate request payloads end-to-end without hitting `graph.facebook.com`.

## What it does

- **`whatsapp-mock`** (default) — runs Prism in **mock** mode: generates responses from the OpenAPI examples/schemas. Rejects any request that doesn't match the spec with a `422`, and enforces the `Authorization: Bearer …` security requirement with a `401`. Perfect for local development, CI integration tests, and validating that our Go clients emit legal payloads.
- **`whatsapp-proxy`** (opt-in, `--profile proxy`) — runs Prism in **proxy** mode: forwards requests to the real Graph API and validates both request *and* response against the spec. Useful when recording live traces for debugging or diff'ing expected vs. actual shapes. Requires real credentials.

## Starting it

```bash
# via Make
make mock-whatsapp-up
make mock-whatsapp-logs     # tail
make mock-whatsapp-down     # stop

# or directly
docker compose -f deploy/mocks/docker-compose.mocks.yml up -d whatsapp-mock
```

Mock listens on **http://localhost:4010**.

## Pointing Linktor at the mock

Every Meta Graph API caller in the codebase (backend, Flows client, template service, etc.) honors the `LINKTOR_GRAPH_API_URL` environment variable via `pkg/graphapi.BaseURL()`. Set it and the whole stack talks to Prism:

```bash
# one-off run
LINKTOR_GRAPH_API_URL=http://localhost:4010 go run ./cmd/server

# for tests
make test-with-mock
```

Default (env unset or empty) is `https://graph.facebook.com` — production behaviour is unchanged.

## What the mock will and won't catch

Catches:
- Wrong URL paths (`/v23.0/{phone_number_id}/message` → 404).
- Missing required body fields (`messaging_product`, `to`, etc.) → 422 with a precise diagnostic.
- Wrong request content type or auth scheme → 415 / 401.
- Response shapes: use `--profile proxy` (see above) to detect when the real Graph API returns a shape our code doesn't expect.

Does NOT catch:
- Business-logic errors Meta only surfaces under real load (rate limits, template review failures, billing rejections).
- Anything outside the scope of the 30k-line spec (e.g. lookaside media download redirects).

## Sanity check

```bash
# expect HTTP 401 (spec requires Bearer auth)
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:4010/v23.0/me

# expect HTTP 422 + a validation payload (our body is missing required fields
# from one branch of the polymorphic messages schema — this IS the mock telling
# us exactly what Meta's spec requires).
curl -s -X POST http://localhost:4010/v23.0/12345/messages \
  -H "Authorization: Bearer fake" \
  -H "Content-Type: application/json" \
  -d '{"messaging_product":"whatsapp","to":"+15551234567","type":"text","text":{"body":"hi"}}' \
  | jq .
```

## Proxy mode (recording real traffic)

```bash
# needs real creds — export them via env first
docker compose -f deploy/mocks/docker-compose.mocks.yml --profile proxy up -d whatsapp-proxy

# then point Linktor at :4011 instead of :4010
LINKTOR_GRAPH_API_URL=http://localhost:4011 go run ./cmd/server
```

Prism will both validate and forward. Any spec drift (new field, wrong type) shows up as a warning in the proxy logs without breaking the request.

## Alternative: code generation

We considered `openapi-generator-cli -g go` to generate a Go client from the spec. Rejected for three reasons:

1. The spec uses heavily polymorphic `oneOf` request bodies for `/{Phone-Number-ID}/messages` — the generated Go types would be a forest of untyped `interface{}` wrappers, worse than what we have by hand.
2. Our existing clients encode business logic (rate limiting, retries, coexistence workflow state) that wouldn't survive regeneration.
3. For *validating* outbound requests, the mock server gives us the same feedback with zero code churn.

Generation might make sense later for a **client SDK** we ship to third-party Linktor integrators. Not now.
