package transaction

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type transactionPayload struct {
	EventID      string         `json:"event_id"`
	Type         string         `json:"type"`
	Transaction  string         `json:"transaction"`
	StartTime    string         `json:"start_timestamp"`
	Timestamp    string         `json:"timestamp"`
	Platform     string         `json:"platform"`
	Release      string         `json:"release"`
	Environment  string         `json:"environment"`
	Contexts     map[string]any `json:"contexts"`
	Tags         any            `json:"tags"`
	Spans        []spanPayload  `json:"spans"`
	Measurements map[string]any `json:"measurements"`
}

type spanPayload struct {
	SpanID       string         `json:"span_id"`
	TraceID      string         `json:"trace_id"`
	ParentSpanID string         `json:"parent_span_id"`
	Op           string         `json:"op"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	StartTime    string         `json:"start_timestamp"`
	Timestamp    string         `json:"timestamp"`
	Data         map[string]any `json:"data"`
}

type traceContext struct {
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id"`
	Op           string `json:"op"`
	Status       string `json:"status"`
}

type transactionInfo struct {
	Source string `json:"source"`
}

func ParseRawMessage(body []byte) (TransactionRecord, []SpanRecord, error) {
	var raw RawEnvelopeItemMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return TransactionRecord{}, nil, fmt.Errorf("decode raw transaction item: %w", err)
	}
	if raw.Item.Type != "transaction" {
		return TransactionRecord{}, nil, nil
	}
	var payload transactionPayload
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return TransactionRecord{}, nil, fmt.Errorf("decode transaction payload: %w", err)
	}
	eventID := firstNonEmpty(payload.EventID, raw.EventID)
	if eventID == "" {
		return TransactionRecord{}, nil, fmt.Errorf("transaction event_id is required")
	}

	started, err := parseTimestamp(payload.StartTime)
	if err != nil {
		return TransactionRecord{}, nil, fmt.Errorf("transaction start_timestamp: %w", err)
	}
	ended, err := parseTimestamp(payload.Timestamp)
	if err != nil {
		return TransactionRecord{}, nil, fmt.Errorf("transaction timestamp: %w", err)
	}
	trace := payload.traceContext()
	info := payload.transactionInfo()
	measurements, _ := json.Marshal(emptyMap(payload.Measurements))
	contexts, _ := json.Marshal(emptyMap(payload.Contexts))
	tags, _ := json.Marshal(payload.Tags)
	if len(tags) == 0 || string(tags) == "null" {
		tags = []byte("{}")
	}

	record := TransactionRecord{
		EventID:         eventID,
		OrganizationID:  raw.OrganizationID,
		ProjectID:       raw.ProjectID,
		ProjectKeyID:    raw.ProjectKeyID,
		TraceID:         trace.TraceID,
		SpanID:          trace.SpanID,
		ParentSpanID:    trace.ParentSpanID,
		TransactionName: strings.TrimSpace(payload.Transaction),
		Source:          firstNonEmpty(info.Source, "custom"),
		Operation:       trace.Op,
		Status:          trace.Status,
		StartTimestamp:  started,
		EndTimestamp:    ended,
		DurationMS:      durationMS(started, ended),
		ReceivedAt:      raw.ReceivedAt,
		Platform:        payload.Platform,
		Release:         payload.Release,
		Environment:     payload.Environment,
		SDKName:         raw.SDKName,
		SDKVersion:      raw.SDKVersion,
		SpanCount:       uint64(len(payload.Spans)),
		Measurements:    string(measurements),
		Contexts:        string(contexts),
		Tags:            string(tags),
		RawTransaction:  raw.Payload,
	}

	spans := make([]SpanRecord, 0, len(payload.Spans))
	for _, span := range payload.Spans {
		spanStart, err := parseTimestamp(span.StartTime)
		if err != nil {
			return TransactionRecord{}, nil, fmt.Errorf("span %s start_timestamp: %w", span.SpanID, err)
		}
		spanEnd, err := parseTimestamp(span.Timestamp)
		if err != nil {
			return TransactionRecord{}, nil, fmt.Errorf("span %s timestamp: %w", span.SpanID, err)
		}
		data, _ := json.Marshal(emptyMap(span.Data))
		spans = append(spans, SpanRecord{
			EventID:        eventID,
			ProjectID:      raw.ProjectID,
			TraceID:        firstNonEmpty(span.TraceID, trace.TraceID),
			SpanID:         span.SpanID,
			ParentSpanID:   span.ParentSpanID,
			Operation:      span.Op,
			Description:    span.Description,
			Status:         span.Status,
			StartTimestamp: spanStart,
			EndTimestamp:   spanEnd,
			DurationMS:     durationMS(spanStart, spanEnd),
			Data:           string(data),
		})
	}
	return record, spans, nil
}

func (p transactionPayload) traceContext() traceContext {
	contexts := p.Contexts
	if contexts == nil {
		return traceContext{}
	}
	raw, _ := json.Marshal(contexts["trace"])
	var trace traceContext
	_ = json.Unmarshal(raw, &trace)
	return trace
}

func (p transactionPayload) transactionInfo() transactionInfo {
	contexts := p.Contexts
	if contexts == nil {
		return transactionInfo{}
	}
	raw, _ := json.Marshal(contexts["transaction_info"])
	var info transactionInfo
	_ = json.Unmarshal(raw, &info)
	return info
}

func parseTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func durationMS(started time.Time, ended time.Time) float64 {
	if started.IsZero() || ended.IsZero() || ended.Before(started) {
		return 0
	}
	return float64(ended.Sub(started).Microseconds()) / 1000
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func emptyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
