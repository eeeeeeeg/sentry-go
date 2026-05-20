package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"sentry-lite/internal/testutil"
)

func TestReleaseFileHeaders(t *testing.T) {
	got := releaseFileHeaders([]string{
		"Content-Type: application/javascript",
		"invalid",
		"Cache-Control: max-age=31536000",
	})

	if got["Content-Type"] != "application/javascript" {
		t.Fatalf("Content-Type = %q", got["Content-Type"])
	}
	if got["Cache-Control"] != "max-age=31536000" {
		t.Fatalf("Cache-Control = %q", got["Cache-Control"])
	}
	if _, ok := got["invalid"]; ok {
		t.Fatal("invalid header was included")
	}
}

func TestReleaseCreateHTTPFixture(t *testing.T) {
	req := testutil.HTTPFixture(t, "requests", "release-create.http")

	if req.Method != http.MethodPost {
		t.Fatalf("Method = %q", req.Method)
	}
	if req.URL.Path != "/api/0/organizations/demo/releases/" {
		t.Fatalf("Path = %q", req.URL.Path)
	}
	if req.Header.Get("Authorization") != "Bearer demo-api-token" {
		t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
	}

	var body createReleaseRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Version != "frontend@1.0.0" {
		t.Fatalf("Version = %q", body.Version)
	}
	if len(body.Projects) != 1 || body.Projects[0] != "web" {
		t.Fatalf("Projects = %#v", body.Projects)
	}
}

func TestDeployCreateHTTPFixture(t *testing.T) {
	req := testutil.HTTPFixture(t, "requests", "deploy-create.http")

	if req.Method != http.MethodPost {
		t.Fatalf("Method = %q", req.Method)
	}
	if req.URL.Path != "/api/0/organizations/demo/releases/frontend@1.0.0/deploys/" {
		t.Fatalf("Path = %q", req.URL.Path)
	}

	var body createDeployRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Environment != "production" {
		t.Fatalf("Environment = %q", body.Environment)
	}
	if body.DateFinished != "2026-05-18T10:00:00Z" {
		t.Fatalf("DateFinished = %q", body.DateFinished)
	}
}

func TestReleaseFileUploadHTTPFixture(t *testing.T) {
	req := testutil.HTTPFixture(t, "requests", "release-file-upload.http")

	if req.Method != http.MethodPost {
		t.Fatalf("Method = %q", req.Method)
	}
	if req.Header.Get("Content-Type") == "" {
		t.Fatal("Content-Type is empty")
	}
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("ParseMultipartForm() error = %v", err)
	}
	if req.FormValue("name") != "~/assets/app.min.js.map" {
		t.Fatalf("name = %q", req.FormValue("name"))
	}
	if req.MultipartForm.Value["header"][0] != "Content-Type: application/json" {
		t.Fatalf("header = %#v", req.MultipartForm.Value["header"])
	}
	file, header, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile() error = %v", err)
	}
	defer file.Close()
	if header.Filename != "app.min.js.map" {
		t.Fatalf("filename = %q", header.Filename)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !json.Valid(content) {
		t.Fatalf("uploaded sourcemap is not valid JSON: %q", string(content))
	}
}
