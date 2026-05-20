package alert

import (
	"time"

	"sentry-lite/internal/normalize"
)

type Event struct {
	Type        string                    `json:"type"`
	ProjectID   string                    `json:"project_id"`
	IssueID     string                    `json:"issue_id"`
	EventID     string                    `json:"event_id"`
	Level       string                    `json:"level"`
	Title       string                    `json:"title"`
	Message     string                    `json:"message"`
	Environment string                    `json:"environment,omitempty"`
	Release     string                    `json:"release,omitempty"`
	OccurredAt  time.Time                 `json:"occurred_at"`
	Event       normalize.NormalizedEvent `json:"event"`
}

type WebhookPayload struct {
	Type        string    `json:"type"`
	ProjectID   string    `json:"project_id"`
	IssueID     string    `json:"issue_id"`
	EventID     string    `json:"event_id"`
	Level       string    `json:"level"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Environment string    `json:"environment,omitempty"`
	Release     string    `json:"release,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
}
