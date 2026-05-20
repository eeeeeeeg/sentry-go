package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type sessionPayload struct {
	SID               string       `json:"sid"`
	DID               string       `json:"did"`
	Seq               float64      `json:"seq"`
	Timestamp         string       `json:"timestamp"`
	Started           string       `json:"started"`
	Init              bool         `json:"init"`
	Duration          float64      `json:"duration"`
	Status            string       `json:"status"`
	Errors            uint64       `json:"errors"`
	AbnormalMechanism string       `json:"abnormal_mechanism"`
	Attrs             sessionAttrs `json:"attrs"`
}

type sessionsPayload struct {
	Aggregates []sessionAggregate `json:"aggregates"`
	Attrs      sessionAttrs       `json:"attrs"`
}

type sessionAttrs struct {
	Release     string `json:"release"`
	Environment string `json:"environment"`
}

type sessionAggregate struct {
	Started   string `json:"started"`
	DID       string `json:"did"`
	Exited    uint64 `json:"exited"`
	Errored   uint64 `json:"errored"`
	Abnormal  uint64 `json:"abnormal"`
	Unhandled uint64 `json:"unhandled"`
	Crashed   uint64 `json:"crashed"`
}

func ParseRawMessage(body []byte) ([]SessionRecord, error) {
	var raw RawEnvelopeItemMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode raw session item: %w", err)
	}
	switch raw.Item.Type {
	case "session":
		return parseSession(raw)
	case "sessions":
		return parseSessionAggregates(raw)
	default:
		return nil, nil
	}
}

func parseSession(raw RawEnvelopeItemMessage) ([]SessionRecord, error) {
	var payload sessionPayload
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode session payload: %w", err)
	}
	if strings.TrimSpace(payload.Attrs.Release) == "" {
		return nil, fmt.Errorf("session attrs.release is required")
	}
	startedAt, err := parseRequiredTime(payload.Started, "session started")
	if err != nil {
		return nil, err
	}
	timestamp := raw.ReceivedAt
	if strings.TrimSpace(payload.Timestamp) != "" {
		timestamp, err = parseFlexibleTime(payload.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("session timestamp: %w", err)
		}
	}
	status := strings.TrimSpace(payload.Status)
	if status == "" {
		status = "ok"
	}
	errorsCount := payload.Errors
	if (status == "crashed" || status == "unhandled") && errorsCount == 0 {
		errorsCount = 1
	}
	sequence := payload.Seq
	if payload.Init {
		sequence = 0
	}
	return []SessionRecord{{
		ProjectID:         raw.ProjectID,
		ProjectKeyID:      raw.ProjectKeyID,
		SessionID:         strings.TrimSpace(payload.SID),
		DistinctID:        hashDistinctID(payload.DID),
		StartedAt:         startedAt,
		Bucket:            startedAt.Truncate(time.Minute),
		Timestamp:         timestamp,
		ReceivedAt:        raw.ReceivedAt,
		Release:           payload.Attrs.Release,
		Environment:       payload.Attrs.Environment,
		Status:            status,
		Init:              payload.Init,
		Sequence:          sequence,
		Errors:            errorsCount,
		Duration:          payload.Duration,
		Quantity:          1,
		AbnormalMechanism: payload.AbnormalMechanism,
		Source:            "session",
		SDKName:           raw.SDKName,
		SDKVersion:        raw.SDKVersion,
	}}, nil
}

func parseSessionAggregates(raw RawEnvelopeItemMessage) ([]SessionRecord, error) {
	var payload sessionsPayload
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode sessions payload: %w", err)
	}
	if strings.TrimSpace(payload.Attrs.Release) == "" {
		return nil, fmt.Errorf("sessions attrs.release is required")
	}
	records := make([]SessionRecord, 0, len(payload.Aggregates))
	for _, aggregate := range payload.Aggregates {
		startedAt, err := parseRequiredTime(aggregate.Started, "aggregate started")
		if err != nil {
			return nil, err
		}
		for _, count := range aggregateCounts(aggregate) {
			if count.quantity == 0 {
				continue
			}
			records = append(records, SessionRecord{
				ProjectID:    raw.ProjectID,
				ProjectKeyID: raw.ProjectKeyID,
				DistinctID:   hashDistinctID(aggregate.DID),
				StartedAt:    startedAt,
				Bucket:       startedAt.Truncate(time.Minute),
				Timestamp:    raw.ReceivedAt,
				ReceivedAt:   raw.ReceivedAt,
				Release:      payload.Attrs.Release,
				Environment:  payload.Attrs.Environment,
				Status:       count.status,
				Quantity:     count.quantity,
				Source:       "sessions",
				SDKName:      raw.SDKName,
				SDKVersion:   raw.SDKVersion,
			})
		}
	}
	return records, nil
}

type aggregateCount struct {
	status   string
	quantity uint64
}

func aggregateCounts(item sessionAggregate) []aggregateCount {
	return []aggregateCount{
		{status: "exited", quantity: item.Exited},
		{status: "errored", quantity: item.Errored},
		{status: "abnormal", quantity: item.Abnormal},
		{status: "unhandled", quantity: item.Unhandled},
		{status: "crashed", quantity: item.Crashed},
	}
}

func parseRequiredTime(value string, field string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("%s is required", field)
	}
	parsed, err := parseFlexibleTime(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", field, err)
	}
	return parsed, nil
}

func parseFlexibleTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func hashDistinctID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
