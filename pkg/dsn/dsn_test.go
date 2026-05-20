package dsn

import "testing"

func TestParse(t *testing.T) {
	got, err := Parse("https://public-key@example.com/project-id")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got.PublicKey != "public-key" {
		t.Fatalf("PublicKey = %q", got.PublicKey)
	}
	if got.ProjectID != "project-id" {
		t.Fatalf("ProjectID = %q", got.ProjectID)
	}
	if got.Host != "example.com" {
		t.Fatalf("Host = %q", got.Host)
	}
}

func TestParseRejectsMissingPublicKey(t *testing.T) {
	_, err := Parse("https://example.com/project-id")
	if err != ErrMissingKey {
		t.Fatalf("Parse() error = %v, want %v", err, ErrMissingKey)
	}
}

func TestParseRejectsNestedProjectPath(t *testing.T) {
	_, err := Parse("https://public-key@example.com/org/project-id")
	if err != ErrMissingProject {
		t.Fatalf("Parse() error = %v, want %v", err, ErrMissingProject)
	}
}
