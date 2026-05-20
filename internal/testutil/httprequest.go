package testutil

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func HTTPFixture(t *testing.T, parts ...string) *http.Request {
	t.Helper()
	raw := Fixture(t, parts...)
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("parse HTTP fixture %v: %v", parts, err)
	}
	body := fixtureHTTPBody(raw)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	return req
}

func fixtureHTTPBody(raw []byte) []byte {
	text := string(raw)
	if _, body, ok := strings.Cut(text, "\r\n\r\n"); ok {
		return []byte(body)
	}
	if _, body, ok := strings.Cut(text, "\n\n"); ok {
		return []byte(body)
	}
	return nil
}
