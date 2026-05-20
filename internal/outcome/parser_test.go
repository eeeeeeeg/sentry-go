package outcome

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseClientReport(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		ReceivedAt:   time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC),
		ProjectID:    "018f3a8b-4214-7c9f-b2f5-7b7a7f534101",
		ProjectKeyID: "018f3a8b-4214-7c9f-b2f5-7b7a7f534102",
		SDKName:      "sentry.javascript.browser",
		Item:         EnvelopeItem{Type: "client_report", Category: "outcome"},
		Payload:      json.RawMessage(`{"timestamp":1779098400,"discarded_events":[{"reason":"sample_rate","category":"error","quantity":2}]}`),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ParseRawMessage(body)
	if err != nil {
		t.Fatalf("ParseRawMessage() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Reason != "sample_rate" || got[0].Category != "error" || got[0].Quantity != 2 {
		t.Fatalf("outcome = %#v", got[0])
	}
	if got[0].Source != "client_report" {
		t.Fatalf("Source = %q", got[0].Source)
	}
}

func TestParseSyntheticOutcome(t *testing.T) {
	raw := RawOutcomeMessage{
		ReceivedAt: time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC),
		ProjectID:  "018f3a8b-4214-7c9f-b2f5-7b7a7f534101",
		Category:   "transaction",
		Reason:     "rate_limited",
		Source:     "server",
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ParseRawMessage(body)
	if err != nil {
		t.Fatalf("ParseRawMessage() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Quantity != 1 {
		t.Fatalf("Quantity = %d", got[0].Quantity)
	}
}
