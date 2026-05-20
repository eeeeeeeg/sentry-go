package replay

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type replayPayload struct {
	ReplayID      string         `json:"replay_id"`
	EventID       string         `json:"event_id"`
	TraceID       string         `json:"trace_id"`
	TransactionID string         `json:"transaction_id"`
	SegmentID     int            `json:"segment_id"`
	Timestamp     any            `json:"timestamp"`
	Started       any            `json:"replay_start_timestamp"`
	Contexts      map[string]any `json:"contexts"`
	Tags          any            `json:"tags"`
	URLs          []string       `json:"urls"`
}

type traceContext struct {
	TraceID string `json:"trace_id"`
}

func ParseRawMessage(body []byte) (ReplayItem, error) {
	var raw RawEnvelopeItemMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return ReplayItem{}, fmt.Errorf("decode raw replay item: %w", err)
	}
	switch raw.Item.Type {
	case "replay_event", "replay_recording":
	default:
		return ReplayItem{}, nil
	}

	payload, _ := parseReplayPayload(raw.Payload)
	replayID := canonicalID(firstNonEmpty(payload.ReplayID, raw.EventID))
	if replayID == "" {
		replayID = hashID(raw.Payload)
	}
	eventID := canonicalID(firstNonEmpty(payload.EventID, raw.EventID))
	traceID := firstNonEmpty(payload.TraceID, payload.traceID())
	timestamp := parseReplayTime(payload.Timestamp)
	if timestamp.IsZero() {
		timestamp = defaultTime(raw.ReceivedAt)
	}
	contentType := strings.TrimSpace(raw.Item.ContentType)
	if contentType == "" {
		contentType = "application/json"
	}
	metadata, _ := json.Marshal(payload)
	return ReplayItem{
		MessageID:      raw.MessageID,
		OrganizationID: raw.OrganizationID,
		ProjectID:      raw.ProjectID,
		ProjectKeyID:   raw.ProjectKeyID,
		ReplayID:       replayID,
		EventID:        eventID,
		TraceID:        traceID,
		TransactionID:  canonicalID(payload.TransactionID),
		SegmentID:      payload.SegmentID,
		ItemType:       raw.Item.Type,
		Timestamp:      timestamp,
		ReceivedAt:     defaultTime(raw.ReceivedAt),
		SDKName:        raw.SDKName,
		SDKVersion:     raw.SDKVersion,
		ContentType:    contentType,
		Size:           int64(len(raw.Payload)),
		Metadata:       metadata,
		Payload:        raw.Payload,
	}, nil
}

func parseReplayPayload(body []byte) (replayPayload, error) {
	var payload replayPayload
	if !json.Valid(body) {
		return payload, fmt.Errorf("payload is not JSON")
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func (p replayPayload) traceID() string {
	if p.Contexts == nil {
		return ""
	}
	raw, _ := json.Marshal(p.Contexts["trace"])
	var trace traceContext
	_ = json.Unmarshal(raw, &trace)
	return trace.TraceID
}

func parseReplayTime(value any) time.Time {
	switch typed := value.(type) {
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return time.Time{}
		}
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed.UTC()
		}
	case float64:
		seconds := int64(typed)
		nanos := int64((typed - float64(seconds)) * 1e9)
		return time.Unix(seconds, nanos).UTC()
	}
	return time.Time{}
}

func defaultTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func canonicalID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) != 32 {
		return value
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
			return value
		}
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", value[0:8], value[8:12], value[12:16], value[16:20], value[20:32])
}

func hashID(value []byte) string {
	sum := sha256.Sum256(value)
	return fmt.Sprintf("%x", sum)
}
