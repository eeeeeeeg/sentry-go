package attachment

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseAttachment(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		MessageID:      "event-attachment-1",
		ReceivedAt:     time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		ProjectID:      "00000000-0000-0000-0000-000000000002",
		ProjectKeyID:   "00000000-0000-0000-0000-000000000003",
		EventID:        "018f3a8b42147c9fb2f57b7a7f534101",
		Item: EnvelopeItem{
			Type:        "attachment",
			Category:    "attachment",
			ContentType: "text/plain",
			Filename:    "log.txt",
			Attachment:  "event.attachment",
		},
		Payload: []byte("hello"),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRawMessage(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Filename != "log.txt" || got.ContentType != "text/plain" || got.Size != 5 {
		t.Fatalf("attachment = %#v", got)
	}
	if string(got.Content) != "hello" {
		t.Fatalf("content = %q", string(got.Content))
	}
}

func TestParseAttachmentRequiresFilename(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		Item:    EnvelopeItem{Type: "attachment"},
		Payload: []byte("hello"),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRawMessage(body); err == nil {
		t.Fatal("ParseRawMessage() error = nil")
	}
}
