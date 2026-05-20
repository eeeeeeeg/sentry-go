# Sentry Compatibility Matrix

This matrix tracks Sentry compatibility work against official Sentry SDK/API behavior.

Status values:

- `done`: Implemented and covered by local tests or fixtures.
- `partial`: Endpoint or item exists, but behavior/schema is incomplete.
- `planned`: Not implemented yet.
- `deferred`: Intentionally not in the current milestone.

## Ingest And Relay

| Area | Official reference | Method / item | Status | Current behavior | Gaps | Fixture |
| --- | --- | --- | --- | --- | --- | --- |
| Envelope ingestion | https://develop.sentry.dev/sdk/envelopes/ | `POST /api/{project_id}/envelope/` | partial | Accepts newline-delimited envelopes, gzip, CORS preflight, Sentry auth headers, Sentry DSN project IDs, and all currently routed envelope categories. | Envelope header validation is incomplete and maximum size/quotas are simplified compared with Relay. | `testdata/sentry-fixtures/envelopes/javascript-error.envelope`, `testdata/sentry-fixtures/envelopes/mixed-client-report-event.envelope` |
| Legacy store ingestion | https://develop.sentry.dev/sdk/overview/ | `POST /api/{project_id}/store/` | partial | Accepts JSON event payload through the same handler as envelope. | Store endpoint is legacy; response/error behavior not fully verified against old SDKs. | `testdata/sentry-fixtures/envelopes/store-event.json` |
| Authentication | https://develop.sentry.dev/sdk/overview/ | Sentry DSN, `X-Sentry-Auth`, `Authorization: Sentry ...`, `sentry_key` | partial | Uses Sentry DSN format `{protocol}://{public_key}@{host}/{project_id}` and extracts public keys from common SDK auth forms. | Does not validate sentry_version, sentry_client, sentry_timestamp, or secret key semantics. | `testdata/sentry-fixtures/requests/envelope.http` |
| Rate limits | https://develop.sentry.dev/sdk/expected-features/rate-limiting/ | `X-Sentry-Rate-Limits`, `Retry-After` | partial | Checks rate limits per project key, IP, and envelope category; event category rejection returns `429`; non-event category rejection drops only that category and emits synthetic outcomes. | Uses one configured limit across all categories. | Not yet |
| Outcomes / client reports | https://develop.sentry.dev/sdk/client-reports/ | `client_report` item | partial | `client_report` items are parsed by `worker-outcome` and written to ClickHouse `sentry.outcomes`; server-side rate limiting emits synthetic outcomes. | No outcomes query API or dashboard yet; only rate limit server discard reason is recorded. | `testdata/sentry-fixtures/envelopes/mixed-client-report-event.envelope` |
| Event payload normalization | https://develop.sentry.dev/sdk/data-model/event-payloads/ | `event` | partial | Supports message events and SDK-style `exception.values[].stacktrace.frames[]`; canonicalizes 32-char event IDs. | Missing full request/user/contexts/modules/threads/mechanism model and complete Sentry validation rules. | `testdata/sentry-fixtures/envelopes/javascript-error.envelope` |
| Transactions | https://develop.sentry.dev/sdk/telemetry/traces/ | `transaction` item | partial | `transaction` items are routed to `worker-transaction`, parsed as Sentry transaction events, written to ClickHouse `sentry.transactions` plus child `sentry.spans`, and exposed through internal transaction list/detail/span APIs. | No Sentry Discover-compatible API, dynamic sampling, trace metrics, span-only ingestion, or transaction/span indexed outcome handling yet. | `testdata/sentry-fixtures/envelopes/transaction.envelope`, `testdata/sentry-fixtures/requests/transactions.http` |
| Sessions | https://develop.sentry.dev/sdk/telemetry/sessions/ | `session`, `sessions` items | partial | `session` and `sessions` items are parsed by `worker-session` and written to ClickHouse `sentry.sessions`; distinct IDs are hashed before storage. | No release health query API yet; terminal-state deduplication and 5-day update window are not enforced. | `testdata/sentry-fixtures/envelopes/sessions.envelope` |
| Attachments | https://docs.sentry.io/platforms/javascript/enriching-events/attachments/ | `attachment` item; `/api/0/projects/{org}/{project}/events/{event_id}/attachments/` | partial | `attachment` items are fully scanned from envelopes, routed to `worker-attachment`, stored in Postgres `event_attachments`, and exposed through list/get/download/delete event attachment APIs guarded by `event:*` scopes. | Uses Postgres bytea instead of object storage; exact public Sentry attachment endpoint schema still needs live comparison; global envelope size limit is lower than Sentry's documented attachment limits. | `testdata/sentry-fixtures/envelopes/event-with-attachment.envelope` |
| Profiles | https://develop.sentry.dev/sdk/telemetry/profiles/ | `profile`, `profile_chunk` items | partial | `profile` and `profile_chunk` items are routed to `worker-profile`, parsed for profile/transaction/trace metadata, and written to ClickHouse `sentry.profiles` with raw profile JSON. | No profile query API, flamegraph processing, chunk reassembly semantics, or object storage offload yet. | `testdata/sentry-fixtures/envelopes/profile.envelope` |
| Replays | https://docs.sentry.io/product/session-replay/ | `replay_event`, `replay_recording` items; `/api/0/projects/{org}/{project}/replays/{replay_id}/recording-segments/` | partial | `replay_event` and `replay_recording` items are routed to `worker-replay`, persisted in Postgres `replay_items`, and exposed through Sentry-style recording segment list/retrieve APIs. | No replay instance list/detail API yet; replay payloads are stored in Postgres bytea instead of object storage; segment reassembly semantics are minimal. | `testdata/sentry-fixtures/envelopes/replay.envelope`, `testdata/sentry-fixtures/requests/replay-segments.http` |

## API Authentication

| Area | Official reference | Method / item | Status | Current behavior | Gaps | Fixture |
| --- | --- | --- | --- | --- | --- | --- |
| Bearer token auth | https://docs.sentry.io/api/auth/ | `Authorization: Bearer <token>` | partial | Hashes bearer token with SHA-256 and checks `api_tokens.token_hash`, expiry, and scopes. | Token creation/list/revoke API not implemented; no user/team membership enforcement. | `testdata/sentry-fixtures/requests/release-create.http` |
| Scope checks | Sentry endpoint docs | Endpoint-specific scopes | partial | Release endpoints enforce documented scopes such as `project:releases`, `project:read`, `project:write`, `project:admin`, `org:ci`. | Other `/api/0/...` endpoints are not yet migrated to this auth layer. | Not yet |

## Releases, Deploys, And Artifacts

| Area | Official reference | Method / item | Status | Current behavior | Gaps | Fixture |
| --- | --- | --- | --- | --- | --- | --- |
| List organization releases | https://docs.sentry.io/api/releases/list-an-organizations-releases/ | `GET /api/0/organizations/{org}/releases/` | partial | Lists releases with Sentry-like response shape. | Cursor pagination and full release fields are incomplete. | Not yet |
| Create organization release | https://docs.sentry.io/api/releases/create-a-new-release-for-an-organization/ | `POST /api/0/organizations/{org}/releases/` | partial | Requires `version` and `projects`; stores `ref`, `url`, `dateReleased`; returns Sentry-like shape. | Commits/refs processing and exact validation errors are missing. | `testdata/sentry-fixtures/requests/release-create.http` |
| Retrieve organization release | https://docs.sentry.io/api/releases/retrieve-an-organizations-release/ | `GET /api/0/organizations/{org}/releases/{version}/` | partial | Returns release details with deploy count and latest deploy. | `status`, `versionInfo`, health fields, commit/deploy detail completeness are incomplete. | Not yet |
| Update organization release | https://docs.sentry.io/api/releases/update-an-organizations-release/ | `PUT /api/0/organizations/{org}/releases/{version}/` | partial | Updates `ref`, `url`, `dateReleased`. | Commits/refs processing is missing. | Not yet |
| Delete organization release | https://docs.sentry.io/api/releases/delete-an-organizations-release/ | `DELETE /api/0/organizations/{org}/releases/{version}/` | done | Deletes release and cascades related rows; returns `204`. | Needs integration fixture. | Not yet |
| List release deploys | https://docs.sentry.io/api/releases/list-a-releases-deploys/ | `GET /api/0/organizations/{org}/releases/{version}/deploys/` | partial | Lists deploy records. | Cursor pagination and exact deploy schema need verification. | Not yet |
| Create deploy | https://docs.sentry.io/api/releases/create-a-deploy/ | `POST /api/0/organizations/{org}/releases/{version}/deploys/` | partial | Requires `environment`; stores optional `name`, `url`, `dateStarted`, `dateFinished`; returns `201`. | Exact response schema needs comparison against Sentry. | `testdata/sentry-fixtures/requests/deploy-create.http` |
| List release files | https://docs.sentry.io/api/releases/list-an-organizations-release-files/ | `GET /api/0/organizations/{org}/releases/{version}/files/` | partial | Lists file metadata. | Cursor pagination missing. | Not yet |
| Upload organization release file | https://docs.sentry.io/api/releases/upload-a-new-organization-release-file/ | `POST /api/0/organizations/{org}/releases/{version}/files/` | partial | Accepts multipart `file`, optional `name`, `dist`, repeated `header`; returns metadata. | Stores content in Postgres instead of object storage; exact duplicate behavior needs verification. | `testdata/sentry-fixtures/requests/release-file-upload.http`, `testdata/sentry-fixtures/artifacts/app.min.js.map` |
| Retrieve release file | https://docs.sentry.io/api/releases/retrieve-an-organization-releases-file/ | `GET /api/0/organizations/{org}/releases/{version}/files/{file_id}/` | partial | Returns metadata; `download=true` returns raw bytes. | Content-Disposition and exact headers need comparison. | Not yet |
| Update release file | https://docs.sentry.io/api/releases/update-an-organization-release-file/ | `PUT /api/0/organizations/{org}/releases/{version}/files/{file_id}/` | partial | Updates `name` and `dist`. | Exact validation behavior needs verification. | Not yet |
| Delete release file | https://docs.sentry.io/api/releases/delete-an-organization-releases-file/ | `DELETE /api/0/organizations/{org}/releases/{version}/files/{file_id}/` | done | Deletes file; returns `204`. | Needs integration fixture. | Not yet |
| Project release files | Sentry project release file docs | `/api/0/projects/{org}/{project}/releases/{version}/files/` | partial | Supports list/upload/get/update/delete with project filter. | Project release list/retrieve endpoints are not implemented. | Not yet |
| Debug files / dSYMs | https://docs.sentry.io/api/projects/upload-a-new-file/ | `POST /api/0/projects/{org}/{project}/files/dsyms/` | planned | Not implemented. | Need debug file upload/list/delete and object storage. | Not yet |
| Artifact bundles / debug IDs | Sentry CLI source map docs | Artifact bundle endpoints | planned | Not implemented. | Required for modern source map workflows. | Not yet |

## Issues And Events API

| Area | Official reference | Method / item | Status | Current behavior | Gaps | Fixture |
| --- | --- | --- | --- | --- | --- | --- |
| Project issues | https://docs.sentry.io/api/events/ | `/api/0/projects/{org}/{project}/issues/` | planned | Internal dashboard issue API exists under `/api/projects/...`. | Need Sentry-compatible schema, filters, cursor pagination, auth scopes. | Not yet |
| Issue detail | https://docs.sentry.io/api/events/ | `/api/0/issues/{issue_id}/` | planned | Internal issue detail is not Sentry-compatible. | Need full issue schema, status updates, assignment/bookmarks. | Not yet |
| Event detail | https://docs.sentry.io/api/events/ | `/api/0/projects/{org}/{project}/events/{event_id}/` | planned | Internal event detail exists under dashboard API. | Need entries schema, contexts, tags, breadcrumbs, request data. | Not yet |
| Event attachments | Sentry attachment API docs | Event attachment endpoints | planned | Not implemented. | Depends on envelope attachment handling and object storage. | Not yet |

## Validation Rules

- Every `done` or `partial` endpoint must link to an official Sentry document.
- Every new endpoint should add at least one fixture under `testdata/sentry-fixtures/`.
- Fixtures should be wired into Go tests when practical; current ingest and release request fixtures are covered by `internal/ingest` and `internal/api` tests.
- Response bodies should prefer Sentry field names over internal field names.
- Scope checks must be copied from the official endpoint documentation.
- If behavior intentionally differs from Sentry, record it in the `Gaps` column.
