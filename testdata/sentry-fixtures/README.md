# Sentry Fixtures

These fixtures are captured or hand-built from official Sentry SDK/API request shapes.

Use them for compatibility tests when adding or changing Sentry-compatible behavior.

## Directories

- `envelopes/`: Raw Sentry envelope or legacy store payloads for ingest parsing tests.
- `requests/`: HTTP-style request fixtures for endpoint-level integration tests.
- `artifacts/`: Release artifact payloads, including source maps and future debug files.

## Local Defaults

The default migration seeds:

- organization: `demo`
- project: `web`
- DSN public key: `demo-public-key`
- API bearer token: `demo-api-token`

## Validation Rule

Every fixture should map back to a row in `docs/SENTRY_COMPATIBILITY_MATRIX.md`.
