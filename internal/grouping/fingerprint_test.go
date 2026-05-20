package grouping

import (
	"testing"

	"sentry-lite/internal/normalize"
)

func TestFingerprintNormalizesNoisyValues(t *testing.T) {
	base := normalize.NormalizedEvent{
		ProjectID:      "project-1",
		Platform:       "javascript",
		ExceptionType:  "TypeError",
		ExceptionValue: "Cannot load user 123456 from https://example.com/a",
	}
	other := base
	other.ExceptionValue = "Cannot load user 987654 from https://example.com/b"

	if Fingerprint(base) != Fingerprint(other) {
		t.Fatal("fingerprint should ignore noisy ids and urls")
	}
}
