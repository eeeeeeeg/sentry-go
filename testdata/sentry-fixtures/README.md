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
- `requests/event-attachments.http`: Project event attachment list and download requests.

## Local Defaults

The default migration seeds:

- organization: `demo`
- project: `web`
- DSN public key: `demo-public-key`
- API bearer token: `demo-api-token`

## Validation Rule

Every fixture should map back to a row in `docs/SENTRY_COMPATIBILITY_MATRIX.md`.
