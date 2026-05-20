package transaction

import "time"

type RawEnvelopeItemMessage struct {
	MessageID      string       `json:"message_id"`
	ReceivedAt     time.Time    `json:"received_at"`
	SDKName        string       `json:"sdk_name,omitempty"`
	SDKVersion     string       `json:"sdk_version,omitempty"`
	OrganizationID string       `json:"organization_id"`
	ProjectID      string       `json:"project_id"`
	ProjectKeyID   string       `json:"project_key_id"`
	EventID        string       `json:"event_id,omitempty"`
	Item           EnvelopeItem `json:"item"`
	Payload        []byte       `json:"payload"`
}

type EnvelopeItem struct {
	Type     string `json:"type"`
	Category string `json:"category"`
	Length   int    `json:"length"`
}

type TransactionRecord struct {
	EventID         string
	OrganizationID  string
	ProjectID       string
	ProjectKeyID    string
	TraceID         string
	SpanID          string
	ParentSpanID    string
	TransactionName string
	Source          string
	Operation       string
	Status          string
	StartTimestamp  time.Time
	EndTimestamp    time.Time
	DurationMS      float64
	ReceivedAt      time.Time
	Platform        string
	Release         string
	Environment     string
	SDKName         string
	SDKVersion      string
	SpanCount       uint64
	Measurements    string
	Contexts        string
	Tags            string
	RawTransaction  []byte
}

type SpanRecord struct {
	EventID        string
	ProjectID      string
	TraceID        string
	SpanID         string
	ParentSpanID   string
	Operation      string
	Description    string
	Status         string
	StartTimestamp time.Time
	EndTimestamp   time.Time
	DurationMS     float64
	Data           string
}
