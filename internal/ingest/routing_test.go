package ingest

import (
	"testing"

	"sentry-lite/internal/config"
)

func TestEnvelopeItemCategory(t *testing.T) {
	tests := map[string]string{
		"event":            "error",
		"transaction":      "transaction",
		"session":          "session",
		"sessions":         "session",
		"attachment":       "attachment",
		"profile":          "profile",
		"profile_chunk":    "profile",
		"replay_event":     "replay",
		"replay_recording": "replay",
		"client_report":    "outcome",
		"check_in":         "monitor",
		"unknown":          "default",
	}

	for itemType, want := range tests {
		if got := envelopeItemCategory(itemType); got != want {
			t.Fatalf("envelopeItemCategory(%q) = %q, want %q", itemType, got, want)
		}
	}
}

func TestSubjectForEnvelopeItem(t *testing.T) {
	handler := &Handler{cfg: config.Config{
		RawTransactionSubject:  "transactions.raw",
		RawSessionSubject:      "sessions.raw",
		RawAttachmentSubject:   "attachments.raw",
		RawProfileSubject:      "profiles.raw",
		RawReplaySubject:       "replays.raw",
		RawOutcomeSubject:      "outcomes.raw",
		UnsupportedItemSubject: "envelopes.unsupported",
	}}

	tests := map[string]string{
		"transaction": "transactions.raw",
		"session":     "sessions.raw",
		"attachment":  "attachments.raw",
		"profile":     "profiles.raw",
		"replay":      "replays.raw",
		"outcome":     "outcomes.raw",
		"default":     "envelopes.unsupported",
		"monitor":     "envelopes.unsupported",
	}

	for category, want := range tests {
		got := handler.subjectForEnvelopeItem(EnvelopeItemMetadata{Category: category})
		if got != want {
			t.Fatalf("subjectForEnvelopeItem(%q) = %q, want %q", category, got, want)
		}
	}
}
