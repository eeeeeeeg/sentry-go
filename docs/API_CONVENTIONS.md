# API Conventions

This document records the current backend API conventions used by the dashboard.

## Pagination

List APIs should return a stable `items + page` response:

```json
{
  "items": [],
  "page": {
    "limit": 20,
    "offset": 0,
    "total": 0
  }
}
```

Rules:

- `items` must be an array, including empty results.
- `limit` and `offset` are non-negative query parameters.
- Handlers should clamp unsafe `limit` values. Current upper bound is usually `100`.
- `total` is the total count after filters are applied.

Current paginated APIs:

```text
GET /api/projects
GET /api/projects/{project_id}/keys
GET /api/projects/{project_id}/issues
GET /api/projects/{project_id}/events
GET /api/projects/{project_id}/alert-deliveries
```

## Time Filters

Time range filters use RFC3339 timestamps:

```text
since=2026-05-18T08:00:00Z
until=2026-05-18T10:00:00Z
```

Rules:

- Invalid timestamps are ignored by current handlers.
- `since` is inclusive.
- `until` is treated as an upper bound by each query implementation.
- Dashboard `datetime-local` fields are converted to RFC3339 before requests.

## Common Query Filters

Issue and event lists support:

```text
level=fatal|error|warning|info
environment=production
release=1.0.0
since=...
until=...
limit=20
offset=0
```

Issue lists additionally support:

```text
status=unresolved|resolved|ignored|all
```

Alert deliveries support:

```text
status=sent|failed|suppressed
limit=20
offset=0
```

Stats APIs support:

```text
environment=production
release=1.0.0
since=...
until=...
limit=10
```

Current stats APIs:

```text
GET /api/projects/{project_id}/stats/trend
GET /api/projects/{project_id}/stats/levels
GET /api/projects/{project_id}/stats/top-issues
GET /api/projects/{project_id}/stats/top-releases
```

## Error Responses

Errors should use JSON:

```json
{
  "error": "project_not_found",
  "message": "optional detail"
}
```

Rules:

- `error` is a stable machine-readable code.
- `message` is optional and intended for operators during local development.
- Frontend response interception prefers `message`, then `error`, then the transport error.

## Ingestion API

Current endpoint:

```text
POST /api/{project_id}/envelope
POST /api/{project_id}/envelope/
POST /api/{project_id}/store
POST /api/{project_id}/store/
```

Required request properties:

- DSN public key must be provided through `X-Sentry-Key`, `sentry_key`, `X-DSN`, `X-Sentry-Auth`, `Authorization: DSN ...`, or `Authorization: Sentry sentry_key=...`.
- Body size must not exceed `MAX_ENVELOPE_BYTES`.
- Body must be either a non-empty JSON event object or a Sentry SDK envelope containing a supported ingest item.
- `Content-Encoding: gzip` is accepted.
- Browser SDK preflight requests are accepted through `OPTIONS`.
- Envelope item types that are not yet fully processed are accepted as raw envelope items; `client_report` items are routed to the outcome worker.

Basic payload validation:

- `event_id`, `message`, `platform`, `release`, and `environment` must be strings when present.
- `level` must be one of `debug`, `info`, `warning`, `error`, `fatal` when present.
- `timestamp` must be an RFC3339 timestamp when present.
- `exception` must be an object when present.
- `exception.type` and `exception.value` must be strings when present.
- `exception.stacktrace` must be an array when present.
- Sentry SDK style `exception.values[].type`, `exception.values[].value`, and `exception.values[].stacktrace.frames` are accepted.

## Event Attachments API

```text
GET /api/0/projects/{organization_slug}/{project_slug}/events/{event_id}/attachments/
GET /api/0/projects/{organization_slug}/{project_slug}/events/{event_id}/attachments/{attachment_id}/
DELETE /api/0/projects/{organization_slug}/{project_slug}/events/{event_id}/attachments/{attachment_id}/
```

- List/get/download require one of `event:read`, `event:write`, or `event:admin`.
- Delete requires `event:admin`.
- `download=true` returns the raw attachment bytes with `Content-Type` and `Content-Disposition`.

## Performance Transaction API

```text
GET /api/projects/{project_id}/transactions
GET /api/transactions/{event_id}
GET /api/transactions/{event_id}/spans
```

- Transaction list supports `limit`, `offset`, `operation`, `environment`, `release`, `query`, `since`, and `until`.
- `event_id` may be either Sentry's 32-character hex ID or canonical UUID format.
- These are internal read APIs over the `sentry.transactions` and `sentry.spans` ClickHouse tables; Sentry Discover-compatible response shapes are not implemented yet.

## Replay Recording Segment API

```text
GET /api/0/projects/{organization_slug}/{project_slug}/replays/{replay_id}/recording-segments/
GET /api/0/projects/{organization_slug}/{project_slug}/replays/{replay_id}/recording-segments/{segment_id}/
```

- Requires one of `project:read`, `project:write`, or `project:admin`.
- The segment detail endpoint returns the raw recording segment bytes with the stored `Content-Type`.
- `replay_id` may be either Sentry's 32-character hex ID or canonical UUID format.
- At least one of `message`, `exception.type`, `exception.value`, `exception.values[].type`, or `exception.values[].value` is required.

Accepted response:

```json
{
  "id": "optional-event-id",
  "status": "accepted"
}
```

## Health And Metrics

System endpoints:

```text
GET /healthz
GET /readyz
GET /metrics
```

`/healthz` and `/readyz` return JSON. `/metrics` returns Prometheus text format.

## Release And Source Map Compatibility

Current Sentry API-compatible release endpoints:

```text
GET  /api/0/organizations/{organization_slug}/releases/
POST /api/0/organizations/{organization_slug}/releases/
GET  /api/0/organizations/{organization_slug}/releases/{version}/
PUT  /api/0/organizations/{organization_slug}/releases/{version}/
DELETE /api/0/organizations/{organization_slug}/releases/{version}/
GET  /api/0/organizations/{organization_slug}/releases/{version}/deploys/
POST /api/0/organizations/{organization_slug}/releases/{version}/deploys/
GET  /api/0/organizations/{organization_slug}/releases/{version}/files/
POST /api/0/organizations/{organization_slug}/releases/{version}/files/
GET  /api/0/organizations/{organization_slug}/releases/{version}/files/{file_id}/
PUT  /api/0/organizations/{organization_slug}/releases/{version}/files/{file_id}/
DELETE /api/0/organizations/{organization_slug}/releases/{version}/files/{file_id}/
GET  /api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/
POST /api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/
GET  /api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/{file_id}/
PUT  /api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/{file_id}/
DELETE /api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/{file_id}/
```

These endpoints are intended for `sentry-cli` and build plugins that upload release artifacts and JavaScript source maps.

- Requests must include `Authorization: Bearer ...`.
- Bearer tokens are matched against `api_tokens.token_hash` using SHA-256.
- Release endpoints enforce Sentry-style scopes from the official docs, such as `project:releases`, `project:read`, `project:write`, `project:admin`, and `org:ci`.
- Local migrations seed `demo-api-token` for the demo organization.
- Release creation accepts JSON with `version`, `projects`, `ref`, and `url`.
- Deploy creation accepts JSON with required `environment` and optional `name`, `url`, `dateStarted`, and `dateFinished`.
- File upload accepts `multipart/form-data` with required `file`, optional `name`, `dist`, and repeated `header` fields.
- File retrieval accepts `download=true` to return the raw uploaded file body instead of metadata.
- Uploaded files are currently stored in Postgres `release_files.content`; this is a compatibility bridge and can be moved to object storage later.
- File size is limited by `MAX_RELEASE_FILE_BYTES`.
