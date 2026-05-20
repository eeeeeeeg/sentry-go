package profile

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

type ProfileRecord struct {
	ProfileID      string
	EventID        string
	OrganizationID string
	ProjectID      string
	ProjectKeyID   string
	TraceID        string
	TransactionID  string
	Transaction    string
	Platform       string
	Version        string
	Release        string
	Environment    string
	ReceivedAt     time.Time
	SDKName        string
	SDKVersion     string
	ItemType       string
	DurationNS     uint64
	SampleCount    uint64
	ThreadCount    uint64
	RawProfile     []byte
}
