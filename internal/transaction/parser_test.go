package transaction

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseTransaction(t *testing.T) {
	raw := RawEnvelopeItemMessage{
		ReceivedAt:     time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		ProjectID:      "00000000-0000-0000-0000-000000000002",
		ProjectKeyID:   "00000000-0000-0000-0000-000000000003",
		Item:           EnvelopeItem{Type: "transaction", Category: "transaction"},
		Payload: []byte(`{
			"event_id":"018f3a8b42147c9fb2f57b7a7f534104",
			"type":"transaction",
			"transaction":"GET /api/users",
			"start_timestamp":"2026-05-20T09:59:59.000Z",
			"timestamp":"2026-05-20T10:00:00.250Z",
			"platform":"javascript",
			"release":"web@1.0.0",
			"environment":"production",
			"contexts":{
				"trace":{"trace_id":"4c79f60c11214eb38604f4ae0781bfb2","span_id":"a2fb4a1d1a96d312","op":"http.server","status":"ok"},
				"transaction_info":{"source":"route"}
			},
			"spans":[{"span_id":"b2fb4a1d1a96d313","parent_span_id":"a2fb4a1d1a96d312","op":"db","description":"SELECT 1","start_timestamp":"2026-05-20T09:59:59.100Z","timestamp":"2026-05-20T09:59:59.200Z","data":{"db.system":"postgresql"}}]
		}`),
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	transaction, spans, err := ParseRawMessage(body)
	if err != nil {
		t.Fatal(err)
	}
	if transaction.EventID != "018f3a8b-4214-7c9f-b2f5-7b7a7f534104" || transaction.TransactionName != "GET /api/users" {
		t.Fatalf("transaction = %#v", transaction)
	}
	if transaction.TraceID != "4c79f60c11214eb38604f4ae0781bfb2" || transaction.Source != "route" || transaction.DurationMS != 1250 {
		t.Fatalf("transaction trace/duration = %#v", transaction)
	}
	if len(spans) != 1 || spans[0].Operation != "db" || spans[0].DurationMS != 100 {
		t.Fatalf("spans = %#v", spans)
	}
}
