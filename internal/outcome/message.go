package outcome

import (
	"encoding/json"
	"time"
)

type RawEnvelopeItemMessage struct {
	MessageID    string          `json:"message_id"`
	ReceivedAt   time.Time       `json:"received_at"`
	SDKName      string          `json:"sdk_name,omitempty"`
	SDKVersion   string          `json:"sdk_version,omitempty"`
	ProjectID    string          `json:"project_id"`
	ProjectKeyID string          `json:"project_key_id"`
	EventID      string          `json:"event_id,omitempty"`
	Item         EnvelopeItem    `json:"item"`
	Payload      json.RawMessage `json:"payload"`
}

type EnvelopeItem struct {
	Type     string `json:"type"`
	Category string `json:"category"`
	Length   int    `json:"length"`
}

type RawOutcomeMessage struct {
	MessageID    string    `json:"message_id"`
	ReceivedAt   time.Time `json:"received_at"`
	SDKName      string    `json:"sdk_name,omitempty"`
	SDKVersion   string    `json:"sdk_version,omitempty"`
	ProjectID    string    `json:"project_id"`
	ProjectKeyID string    `json:"project_key_id"`
	EventID      string    `json:"event_id,omitempty"`
	Category     string    `json:"category"`
	Reason       string    `json:"reason"`
	Quantity     uint64    `json:"quantity"`
	Source       string    `json:"source"`
}

type Outcome struct {
	ProjectID    string
	ProjectKeyID string
	EventID      string
	Timestamp    time.Time
	ReceivedAt   time.Time
	Category     string
	Reason       string
	Quantity     uint64
	Source       string
	SDKName      string
	SDKVersion   string
}
