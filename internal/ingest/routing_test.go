package ingest

import (
	"net/http/httptest"
	"testing"
	"time"

	"sentry-lite/internal/config"
	"sentry-lite/internal/quota"
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

func TestEnvelopeCategoriesDeduplicates(t *testing.T) {
	got := envelopeCategories([]EnvelopeItem{
		{EnvelopeItemMetadata: EnvelopeItemMetadata{Category: "error"}},
		{EnvelopeItemMetadata: EnvelopeItemMetadata{Category: "outcome"}},
		{EnvelopeItemMetadata: EnvelopeItemMetadata{Category: "error"}},
	})
	if len(got) != 2 || got[0] != "error" || got[1] != "outcome" {
		t.Fatalf("envelopeCategories() = %#v", got)
	}
}

func TestFilterAllowedItemsKeepsAllowedEvent(t *testing.T) {
	decoded := decodedEnvelope{
		HasEvent: true,
		Items: []EnvelopeItem{
			{
				EnvelopeItemMetadata: EnvelopeItemMetadata{Type: "client_report", Category: "outcome"},
				Payload:              []byte(`{"discarded_events":[]}`),
			},
			{
				EnvelopeItemMetadata: EnvelopeItemMetadata{Type: "event", Category: "error"},
				Payload:              []byte(`{"message":"boom"}`),
			},
		},
	}
	filtered := decoded.filterAllowedItems(categoryRateResults{
		"outcome": {Category: "outcome", Result: quota.Result{Allowed: false}},
		"error":   {Category: "error", Result: quota.Result{Allowed: true}},
	})

	if !filtered.HasEvent {
		t.Fatal("HasEvent = false")
	}
	if len(filtered.Items) != 1 || filtered.Items[0].Type != "event" {
		t.Fatalf("Items = %#v", filtered.Items)
	}
}

func TestEventRateLimited(t *testing.T) {
	if !eventRateLimited(categoryRateResults{
		"error": {Category: "error", Result: quota.Result{Allowed: false}},
	}) {
		t.Fatal("eventRateLimited() = false")
	}
	if eventRateLimited(categoryRateResults{
		"outcome": {Category: "outcome", Result: quota.Result{Allowed: false}},
	}) {
		t.Fatal("eventRateLimited() = true")
	}
}

func TestWriteSentryRateLimitHeadersUsesCategory(t *testing.T) {
	rec := httptest.NewRecorder()
	writeSentryRateLimitHeaders(rec, categoryRateResult{
		Category: "transaction",
		Result: quota.Result{
			Allowed:    false,
			ResetAfter: 30 * time.Second,
		},
	})

	if got := rec.Header().Get("Retry-After"); got != "30" {
		t.Fatalf("Retry-After = %q", got)
	}
	if got := rec.Header().Get("X-Sentry-Rate-Limits"); got != "30:transaction:project:rate_limited" {
		t.Fatalf("X-Sentry-Rate-Limits = %q", got)
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
