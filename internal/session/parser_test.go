package session

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseSession(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		ReceivedAt:   time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		ProjectID:    "00000000-0000-0000-0000-000000000001",
		ProjectKeyID: "00000000-0000-0000-0000-000000000002",
		Item:         EnvelopeItem{Type: "session", Category: "session"},
		Payload:      json.RawMessage(`{"sid":"7c7b6585-f901-4351-bf8d-02711b721929","did":"user-1","init":true,"started":"2026-05-20T09:58:00Z","status":"crashed","attrs":{"release":"web@1.0.0","environment":"production"}}`),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRawMessage(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("records = %d", len(got))
	}
	if got[0].Status != "crashed" || got[0].Errors != 1 || got[0].Quantity != 1 || got[0].Release != "web@1.0.0" {
		t.Fatalf("record = %#v", got[0])
	}
	if got[0].DistinctID == "" || got[0].DistinctID == "user-1" {
		t.Fatalf("distinct id was not hashed: %q", got[0].DistinctID)
	}
}

func TestParseSessionsAggregate(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		ReceivedAt:   time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		ProjectID:    "00000000-0000-0000-0000-000000000001",
		ProjectKeyID: "00000000-0000-0000-0000-000000000002",
		Item:         EnvelopeItem{Type: "sessions", Category: "session"},
		Payload:      json.RawMessage(`{"aggregates":[{"started":"2026-05-20T09:59:22Z","exited":4,"errored":2,"crashed":1}],"attrs":{"release":"api@2.0.0","environment":"staging"}}`),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRawMessage(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("records = %#v", got)
	}
	if got[0].Bucket.Second() != 0 || got[0].Source != "sessions" {
		t.Fatalf("record = %#v", got[0])
	}
	if got[1].Status != "errored" || got[1].Quantity != 2 {
		t.Fatalf("errored record = %#v", got[1])
	}
}
