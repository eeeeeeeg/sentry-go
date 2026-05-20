package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Fixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	root := repoRoot(t)
	pathParts := append([]string{root, "testdata", "sentry-fixtures"}, parts...)
	body, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("read fixture %v: %v", parts, err)
	}
	return body
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate fixture helper")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
