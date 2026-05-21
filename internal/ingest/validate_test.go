package ingest

import "testing"

func TestValidatePayloadAcceptsMessageEvent(t *testing.T) {
	body := []byte(`{
		"event_id": "018f3a8b-4214-7c9f-b2f5-7b7a7f534101",
		"timestamp": "2026-05-18T10:00:00Z",
		"level": "error",
		"message": "Cannot read properties of undefined",
		"platform": "javascript"
	}`)

	if err := validatePayload(body); err != nil {
		t.Fatalf("validatePayload() error = %v", err)
	}
}

func TestValidatePayloadAcceptsNumericTimestamp(t *testing.T) {
	body := []byte(`{
		"event_id": "018f3a8b42147c9fb2f57b7a7f534101",
		"timestamp": 1779098400.123,
		"level": "error",
		"message": "Cannot read properties of undefined",
		"platform": "javascript"
	}`)

	if err := validatePayload(body); err != nil {
		t.Fatalf("validatePayload() error = %v", err)
	}
}

func TestValidatePayloadAcceptsExceptionEvent(t *testing.T) {
	body := []byte(`{
		"level": "fatal",
		"exception": {
			"type": "TypeError",
			"value": "Cannot read properties of undefined",
			"stacktrace": []
		}
	}`)

	if err := validatePayload(body); err != nil {
		t.Fatalf("validatePayload() error = %v", err)
	}
}

func TestValidatePayloadAcceptsSentrySDKExceptionValues(t *testing.T) {
	body := []byte(`{
		"level": "error",
		"exception": {
			"values": [
				{
					"type": "TypeError",
					"value": "Cannot read properties of undefined",
					"stacktrace": {"frames": []}
				}
			]
		}
	}`)

	if err := validatePayload(body); err != nil {
		t.Fatalf("validatePayload() error = %v", err)
	}
}

func TestValidatePayloadRejectsArray(t *testing.T) {
	if err := validatePayload([]byte(`[]`)); err == nil {
		t.Fatal("validatePayload() expected error for array body")
	}
}

func TestValidatePayloadRejectsInvalidLevel(t *testing.T) {
	if err := validatePayload([]byte(`{"level":"panic","message":"boom"}`)); err == nil {
		t.Fatal("validatePayload() expected error for invalid level")
	}
}

func TestValidatePayloadRejectsInvalidTimestamp(t *testing.T) {
	if err := validatePayload([]byte(`{"timestamp":"not-a-time","message":"boom"}`)); err == nil {
		t.Fatal("validatePayload() expected error for invalid timestamp")
	}
}

func TestValidatePayloadRequiresMessageOrException(t *testing.T) {
	if err := validatePayload([]byte(`{"level":"error"}`)); err == nil {
		t.Fatal("validatePayload() expected error without message or exception")
	}
}

func TestValidatePayloadRejectsInvalidStacktrace(t *testing.T) {
	if err := validatePayload([]byte(`{"exception":{"type":"TypeError","stacktrace":{}}}`)); err == nil {
		t.Fatal("validatePayload() expected error for invalid stacktrace")
	}
}
