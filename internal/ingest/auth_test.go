package ingest

import (
	"net/http"
	"testing"
)

func TestExtractPublicKeyFromSentryAuth(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/web/envelope", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", `Sentry sentry_version=7, sentry_key="abc123"`)

	got, err := extractPublicKey(req)
	if err != nil {
		t.Fatalf("extractPublicKey() error = %v", err)
	}
	if got != "abc123" {
		t.Fatalf("extractPublicKey() = %q", got)
	}
}

func TestExtractPublicKeyFromXSentryAuth(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/web/store", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Sentry-Auth", `Sentry sentry_version=7, sentry_key=abc123, sentry_client=sentry.go/0.37.0`)

	got, err := extractPublicKey(req)
	if err != nil {
		t.Fatalf("extractPublicKey() error = %v", err)
	}
	if got != "abc123" {
		t.Fatalf("extractPublicKey() = %q", got)
	}
}

func TestExtractPublicKeyFromDSNHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/web/envelope", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-DSN", "https://public-key@example.com/web")

	got, err := extractPublicKey(req)
	if err != nil {
		t.Fatalf("extractPublicKey() error = %v", err)
	}
	if got != "public-key" {
		t.Fatalf("extractPublicKey() = %q", got)
	}
}
