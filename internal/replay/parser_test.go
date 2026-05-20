package replay

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseReplayEvent(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		MessageID:      "replay-event-1",
		ReceivedAt:     time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		ProjectID:      "00000000-0000-0000-0000-000000000002",
		ProjectKeyID:   "00000000-0000-0000-0000-000000000003",
		EventID:        "018f3a8b42147c9fb2f57b7a7f534105",
		Item:           EnvelopeItem{Type: "replay_event", Category: "replay", ContentType: "application/json"},
		Payload: []byte(`{
			"replay_id":"018f3a8b42147c9fb2f57b7a7f534105",
			"segment_id":0,
			"timestamp":"2026-05-20T10:00:00Z",
			"contexts":{"trace":{"trace_id":"4c79f60c11214eb38604f4ae0781bfb2"}}
		}`),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRawMessage(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReplayID != "018f3a8b-4214-7c9f-b2f5-7b7a7f534105" || got.TraceID != "4c79f60c11214eb38604f4ae0781bfb2" {
		t.Fatalf("replay = %#v", got)
	}
	if got.ItemType != "replay_event" || got.SegmentID != 0 || got.Size == 0 {
		t.Fatalf("metadata = %#v", got)
	}
}

func TestParseReplayRecording(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		MessageID:      "replay-recording-1",
		ReceivedAt:     time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		ProjectID:      "00000000-0000-0000-0000-000000000002",
		ProjectKeyID:   "00000000-0000-0000-0000-000000000003",
		EventID:        "018f3a8b42147c9fb2f57b7a7f534105",
		Item:           EnvelopeItem{Type: "replay_recording", Category: "replay"},
		Payload:        []byte(`[{"type":5,"timestamp":1747735200,"data":{"tag":"breadcrumb","payload":{"message":"clicked"}}}]`),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRawMessage(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReplayID != "018f3a8b-4214-7c9f-b2f5-7b7a7f534105" || got.ItemType != "replay_recording" {
		t.Fatalf("replay = %#v", got)
	}
	if string(got.Payload) != string(raw.Payload) {
		t.Fatalf("payload = %q", string(got.Payload))
	}
}
