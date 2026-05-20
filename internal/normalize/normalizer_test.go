package normalize

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeSentrySDKExceptionValues(t *testing.T) {
	raw := RawEventMessage{
		ReceivedAt:     time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC),
		OrganizationID: "org-1",
		ProjectID:      "project-1",
		ProjectKeyID:   "key-1",
		Payload: json.RawMessage(`{
			"event_id": "018f3a8b42147c9fb2f57b7a7f534101",
			"timestamp": "2026-05-18T10:00:00Z",
			"platform": "go",
			"sdk": {"name": "sentry.go", "version": "0.37.0"},
			"exception": {
				"values": [
					{"type": "TypeError", "value": "Cannot read properties of undefined"}
				]
			}
		}`),
	}

	got, err := NewNormalizer().Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if got.SDKName != "sentry.go" {
		t.Fatalf("SDKName = %q", got.SDKName)
	}
	if got.EventID != "018f3a8b-4214-7c9f-b2f5-7b7a7f534101" {
		t.Fatalf("EventID = %q", got.EventID)
	}
	if got.ExceptionType != "TypeError" {
		t.Fatalf("ExceptionType = %q", got.ExceptionType)
	}
	if got.Message != "Cannot read properties of undefined" {
		t.Fatalf("Message = %q", got.Message)
	}
}
