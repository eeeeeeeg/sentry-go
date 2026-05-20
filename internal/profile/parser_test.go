package profile

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseProfile(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		ReceivedAt:     time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		ProjectID:      "00000000-0000-0000-0000-000000000002",
		ProjectKeyID:   "00000000-0000-0000-0000-000000000003",
		EventID:        "018f3a8b42147c9fb2f57b7a7f534104",
		Item:           EnvelopeItem{Type: "profile", Category: "profile"},
		Payload: []byte(`{
			"profile_id":"profile-1",
			"transaction_id":"018f3a8b42147c9fb2f57b7a7f534104",
			"transaction_name":"GET /api/users",
			"platform":"javascript",
			"version":"1",
			"release":"web@1.0.0",
			"environment":"production",
			"duration_ns":1200000000,
			"profile":{"samples":[{"elapsed_since_start_ns":"0"},{"elapsed_since_start_ns":"1000000"}],"threads":[{"id":1,"name":"main"}]},
			"transaction":{"trace_id":"4c79f60c11214eb38604f4ae0781bfb2"}
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
	if got.ProfileID != "profile-1" || got.Transaction != "GET /api/users" || got.SampleCount != 2 || got.ThreadCount != 1 {
		t.Fatalf("profile = %#v", got)
	}
	if got.TransactionID != "018f3a8b-4214-7c9f-b2f5-7b7a7f534104" {
		t.Fatalf("TransactionID = %q", got.TransactionID)
	}
}
