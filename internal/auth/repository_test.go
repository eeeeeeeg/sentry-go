package auth

import "testing"

func TestBearerToken(t *testing.T) {
	got, err := bearerToken("Bearer demo-api-token")
	if err != nil {
		t.Fatalf("bearerToken() error = %v", err)
	}
	if got != "demo-api-token" {
		t.Fatalf("bearerToken() = %q", got)
	}
}

func TestBearerTokenRejectsMissingToken(t *testing.T) {
	if _, err := bearerToken("Bearer "); err == nil {
		t.Fatal("bearerToken() expected error")
	}
}

func TestHashToken(t *testing.T) {
	got := hashToken("demo-api-token")
	want := "9477b34a9c255f76f79d282640e9f9d02f1b32a370408fdac63538ce33a788ed"
	if got != want {
		t.Fatalf("hashToken() = %q", got)
	}
}

func TestHasAnyScope(t *testing.T) {
	if !hasAnyScope([]string{"project:read", "project:releases"}, "project:releases") {
		t.Fatal("hasAnyScope() = false")
	}
	if hasAnyScope([]string{"project:read"}, "project:releases") {
		t.Fatal("hasAnyScope() = true")
	}
}
