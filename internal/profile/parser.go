package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type profilePayload struct {
	ProfileID       string         `json:"profile_id"`
	EventID         string         `json:"event_id"`
	TransactionID   string         `json:"transaction_id"`
	Transaction     string         `json:"transaction_name"`
	Platform        string         `json:"platform"`
	Version         string         `json:"version"`
	Release         string         `json:"release"`
	Environment     string         `json:"environment"`
	DurationNS      uint64         `json:"duration_ns"`
	Profile         profileBody    `json:"profile"`
	Measurements    map[string]any `json:"measurements"`
	TransactionMeta map[string]any `json:"transaction"`
}

type profileBody struct {
	Samples []any           `json:"samples"`
	Threads []profileThread `json:"threads"`
}

type profileThread struct {
	ID   any    `json:"id"`
	Name string `json:"name"`
}

func ParseRawMessage(body []byte) (ProfileRecord, error) {
	var raw RawEnvelopeItemMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return ProfileRecord{}, fmt.Errorf("decode raw profile item: %w", err)
	}
	switch raw.Item.Type {
	case "profile", "profile_chunk":
	default:
		return ProfileRecord{}, nil
	}

	var payload profilePayload
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return ProfileRecord{}, fmt.Errorf("decode profile payload: %w", err)
	}
	profileID := firstNonEmpty(payload.ProfileID, payloadString(payload.TransactionMeta, "profile_id"))
	if profileID == "" {
		profileID = hashID(raw.Payload)
	}
	eventID := canonicalEventID(firstNonEmpty(payload.EventID, raw.EventID))
	transactionID := canonicalEventID(firstNonEmpty(payload.TransactionID, payloadString(payload.TransactionMeta, "id")))
	transactionName := firstNonEmpty(payload.Transaction, payloadString(payload.TransactionMeta, "name"))
	traceID := firstNonEmpty(payloadString(payload.TransactionMeta, "trace_id"), payloadString(payload.Measurements, "trace_id"))
	return ProfileRecord{
		ProfileID:      profileID,
		EventID:        eventID,
		OrganizationID: raw.OrganizationID,
		ProjectID:      raw.ProjectID,
		ProjectKeyID:   raw.ProjectKeyID,
		TraceID:        traceID,
		TransactionID:  transactionID,
		Transaction:    transactionName,
		Platform:       payload.Platform,
		Version:        payload.Version,
		Release:        payload.Release,
		Environment:    payload.Environment,
		ReceivedAt:     defaultTime(raw.ReceivedAt),
		SDKName:        raw.SDKName,
		SDKVersion:     raw.SDKVersion,
		ItemType:       raw.Item.Type,
		DurationNS:     payload.DurationNS,
		SampleCount:    uint64(len(payload.Profile.Samples)),
		ThreadCount:    uint64(len(payload.Profile.Threads)),
		RawProfile:     raw.Payload,
	}, nil
}

func payloadString(value map[string]any, key string) string {
	if value == nil {
		return ""
	}
	raw, ok := value[key]
	if !ok {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func hashID(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func defaultTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func canonicalEventID(value string) string {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
