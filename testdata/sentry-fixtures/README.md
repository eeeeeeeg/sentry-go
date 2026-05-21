# Sentry Fixtures

These fixtures are captured or hand-built from official Sentry SDK/API request shapes.

Use them for compatibility tests when adding or changing Sentry-compatible behavior.

## Directories

- `envelopes/`: Raw Sentry envelope or legacy store payloads for ingest parsing tests.
- `requests/`: HTTP-style request fixtures for endpoint-level integration tests.
- `artifacts/`: Release artifact payloads, including source maps and future debug files.

Current envelope fixtures:

- `javascript-error.envelope`: JavaScript SDK style error event envelope.
- `event-with-attachment.envelope`: Event followed by an `attachment` item.
- `mixed-client-report-event.envelope`: Envelope with a `client_report` item followed by an event item.
- `sessions.envelope`: Envelope with individual `session` and aggregate `sessions` items.
- `transaction.envelope`: Performance transaction item with a child span.
- `profile.envelope`: Transaction profile item associated with a transaction.
- `replay.envelope`: Session Replay metadata and recording segment items.
- `requests/event-attachments.http`: Project event attachment list and download requests.
- `requests/transactions.http`: Internal performance transaction list/detail/span requests.
- `requests/replay-segments.http`: Sentry-style Replay recording segment list/retrieve requests.

## Local Defaults

The default migration seeds:

- organization: `demo`
- project: `web`
- Sentry project ID: `1`
- DSN public key: `0123456789abcdef0123456789abcdef`
- public DSN: `http://0123456789abcdef0123456789abcdef@localhost:8080/1`
- API bearer token: `demo-api-token`

## Validation Rule

Every fixture should map back to a row in `docs/SENTRY_COMPATIBILITY_MATRIX.md`.
