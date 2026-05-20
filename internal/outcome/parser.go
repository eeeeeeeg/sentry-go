package outcome

import (
	"encoding/json"
	"fmt"
	"time"
)

type clientReport struct {
	Timestamp       float64          `json:"timestamp"`
	DiscardedEvents []discardedEvent `json:"discarded_events"`
}

type discardedEvent struct {
	Reason   string `json:"reason"`
	Category string `json:"category"`
	Quantity uint64 `json:"quantity"`
}

func ParseRawMessage(body []byte) ([]Outcome, error) {
	var synthetic RawOutcomeMessage
	if err := json.Unmarshal(body, &synthetic); err == nil && synthetic.Reason != "" {
		return []Outcome{outcomeFromSynthetic(synthetic)}, nil
	}

	var raw RawEnvelopeItemMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode raw outcome item: %w", err)
	}
	if raw.Item.Type != "client_report" {
		return nil, nil
	}
	return parseClientReport(raw)
}

func parseClientReport(raw RawEnvelopeItemMessage) ([]Outcome, error) {
	var report clientReport
	if err := json.Unmarshal(raw.Payload, &report); err != nil {
		return nil, fmt.Errorf("decode client report: %w", err)
	}

	timestamp := raw.ReceivedAt
	if report.Timestamp > 0 {
		seconds := int64(report.Timestamp)
		nanos := int64((report.Timestamp - float64(seconds)) * 1e9)
		timestamp = time.Unix(seconds, nanos).UTC()
	}

	items := make([]Outcome, 0, len(report.DiscardedEvents))
	for _, discarded := range report.DiscardedEvents {
		category := discarded.Category
		if category == "" {
			category = "default"
		}
		quantity := discarded.Quantity
		if quantity == 0 {
			quantity = 1
		}
		items = append(items, Outcome{
			ProjectID:    raw.ProjectID,
			ProjectKeyID: raw.ProjectKeyID,
			EventID:      raw.EventID,
			Timestamp:    timestamp,
			ReceivedAt:   raw.ReceivedAt,
			Category:     category,
			Reason:       discarded.Reason,
			Quantity:     quantity,
			Source:       "client_report",
			SDKName:      raw.SDKName,
			SDKVersion:   raw.SDKVersion,
		})
	}
	return items, nil
}

func outcomeFromSynthetic(raw RawOutcomeMessage) Outcome {
	timestamp := raw.ReceivedAt
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	quantity := raw.Quantity
	if quantity == 0 {
		quantity = 1
	}
	return Outcome{
		ProjectID:    raw.ProjectID,
		ProjectKeyID: raw.ProjectKeyID,
		EventID:      raw.EventID,
		Timestamp:    timestamp,
		ReceivedAt:   timestamp,
		Category:     raw.Category,
		Reason:       raw.Reason,
		Quantity:     quantity,
		Source:       raw.Source,
		SDKName:      raw.SDKName,
		SDKVersion:   raw.SDKVersion,
	}
}
