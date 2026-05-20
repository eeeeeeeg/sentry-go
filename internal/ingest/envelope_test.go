package ingest

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"testing"

	"sentry-lite/internal/testutil"
)

func TestDecodeIngestPayloadAcceptsSentryEnvelope(t *testing.T) {
	event := `{"event_id":"018f3a8b421447c9fb2f57b7a7f534101","timestamp":"2026-05-18T10:00:00Z","level":"error","platform":"go","exception":{"values":[{"type":"Error","value":"boom","stacktrace":{"frames":[]}}]}}`
	body := fmt.Sprintf(
		"{\"event_id\":\"018f3a8b421447c9fb2f57b7a7f534101\",\"sdk\":{\"name\":\"sentry.go\",\"version\":\"0.37.0\"}}\n{\"type\":\"event\",\"length\":%d}\n%s\n",
		len(event),
		event,
	)

	got, err := decodeIngestPayload([]byte(body))
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if got.EventID != "018f3a8b421447c9fb2f57b7a7f534101" {
		t.Fatalf("EventID = %q", got.EventID)
	}
	if got.SDKName != "sentry.go" {
		t.Fatalf("SDKName = %q", got.SDKName)
	}
	if string(got.Payload) != event {
		t.Fatalf("Payload = %q", string(got.Payload))
	}
}

func TestDecodeIngestPayloadAcceptsLegacyJSONEvent(t *testing.T) {
	event := []byte(`{"message":"boom","level":"error"}`)

	got, err := decodeIngestPayload(event)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if string(got.Payload) != string(event) {
		t.Fatalf("Payload = %q", string(got.Payload))
	}
}

func TestDecodeIngestPayloadAcceptsEnvelopeFixture(t *testing.T) {
	body := testutil.Fixture(t, "envelopes", "javascript-error.envelope")

	got, err := decodeIngestPayload(body)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if !got.HasEvent {
		t.Fatal("decodeIngestPayload() did not detect event item")
	}
	if got.EventID != "018f3a8b42147c9fb2f57b7a7f534101" {
		t.Fatalf("EventID = %q", got.EventID)
	}
	if got.SDKName != "sentry.javascript.browser" {
		t.Fatalf("SDKName = %q", got.SDKName)
	}
}

func TestDecodeIngestPayloadAcceptsStoreFixture(t *testing.T) {
	body := testutil.Fixture(t, "envelopes", "store-event.json")

	got, err := decodeIngestPayload(body)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if !got.HasEvent {
		t.Fatal("decodeIngestPayload() did not detect JSON event")
	}
	if got.EventID != "018f3a8b42147c9fb2f57b7a7f534102" {
		t.Fatalf("EventID = %q", got.EventID)
	}
}

func TestDecodeIngestPayloadRecordsAllEnvelopeItems(t *testing.T) {
	body := testutil.Fixture(t, "envelopes", "mixed-client-report-event.envelope")

	got, err := decodeIngestPayload(body)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if !got.HasEvent {
		t.Fatal("decodeIngestPayload() did not detect event item")
	}
	if len(got.Items) != 2 {
		t.Fatalf("Items length = %d", len(got.Items))
	}
	if got.Items[0].Type != "client_report" || got.Items[0].Length != 70 {
		t.Fatalf("first item = %#v", got.Items[0])
	}
	if got.Items[0].Category != "outcome" {
		t.Fatalf("first category = %q", got.Items[0].Category)
	}
	if got.Items[1].Type != "event" || got.Items[1].Length != 209 {
		t.Fatalf("second item = %#v", got.Items[1])
	}
	if got.Items[1].Category != "error" {
		t.Fatalf("second category = %q", got.Items[1].Category)
	}
}

func TestDecodeIngestPayloadKeepsItemsAfterEvent(t *testing.T) {
	body := testutil.Fixture(t, "envelopes", "event-with-attachment.envelope")

	got, err := decodeIngestPayload(body)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if !got.HasEvent {
		t.Fatal("decodeIngestPayload() did not detect event item")
	}
	if len(got.Items) != 2 {
		t.Fatalf("Items length = %d", len(got.Items))
	}
	attachment := got.Items[1]
	if attachment.Type != "attachment" || attachment.Category != "attachment" {
		t.Fatalf("attachment item = %#v", attachment)
	}
	if attachment.Filename != "debug.log" || attachment.ContentType != "text/plain" || attachment.Attachment != "event.attachment" {
		t.Fatalf("attachment metadata = %#v", attachment.EnvelopeItemMetadata)
	}
	if string(attachment.Payload) != "hello world" {
		t.Fatalf("attachment payload = %q", string(attachment.Payload))
	}
}

func TestDecodeIngestPayloadAcceptsEnvelopeWithoutEvent(t *testing.T) {
	body := []byte("{}\n{\"type\":\"client_report\"}\n{}\n")

	got, err := decodeIngestPayload(body)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if got.HasEvent {
		t.Fatal("decodeIngestPayload() marked client_report envelope as event")
	}
}

func TestDecodeIngestPayloadAcceptsSessionsFixture(t *testing.T) {
	body := testutil.Fixture(t, "envelopes", "sessions.envelope")

	got, err := decodeIngestPayload(body)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if got.HasEvent {
		t.Fatal("decodeIngestPayload() marked sessions envelope as event")
	}
	if len(got.Items) != 2 {
		t.Fatalf("Items length = %d", len(got.Items))
	}
	if got.Items[0].Type != "session" || got.Items[0].Category != "session" {
		t.Fatalf("first item = %#v", got.Items[0])
	}
	if got.Items[1].Type != "sessions" || got.Items[1].Category != "session" {
		t.Fatalf("second item = %#v", got.Items[1])
	}
}

func TestDecodeIngestPayloadAcceptsTransactionFixture(t *testing.T) {
	body := testutil.Fixture(t, "envelopes", "transaction.envelope")

	got, err := decodeIngestPayload(body)
	if err != nil {
		t.Fatalf("decodeIngestPayload() error = %v", err)
	}
	if got.HasEvent {
		t.Fatal("decodeIngestPayload() marked transaction as error event")
	}
	if got.EventID != "018f3a8b42147c9fb2f57b7a7f534104" {
		t.Fatalf("EventID = %q", got.EventID)
	}
	if len(got.Items) != 1 || got.Items[0].Type != "transaction" || got.Items[0].Category != "transaction" {
		t.Fatalf("Items = %#v", got.Items)
	}
}

func TestReadLimitedRequestBodyAcceptsGzip(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, _ = writer.Write([]byte(`{"message":"boom"}`))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/api/web/envelope", &compressed)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Encoding", "gzip")

	got, err := readLimitedRequestBody(req, 1024)
	if err != nil {
		t.Fatalf("readLimitedRequestBody() error = %v", err)
	}
	if string(got) != `{"message":"boom"}` {
		t.Fatalf("readLimitedRequestBody() = %q", string(got))
	}
}
