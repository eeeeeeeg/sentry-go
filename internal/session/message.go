package session

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
	Item         EnvelopeItem    `json:"item"`
	Payload      json.RawMessage `json:"payload"`
}

type EnvelopeItem struct {
	Type     string `json:"type"`
	Category string `json:"category"`
	Length   int    `json:"length"`
}

type SessionRecord struct {
	ProjectID         string
	ProjectKeyID      string
	SessionID         string
	DistinctID        string
	StartedAt         time.Time
	Bucket            time.Time
	Timestamp         time.Time
	ReceivedAt        time.Time
	Release           string
	Environment       string
	Status            string
	Init              bool
	Sequence          float64
	Errors            uint64
	Duration          float64
	Quantity          uint64
	AbnormalMechanism string
	Source            string
	SDKName           string
	SDKVersion        string
}
