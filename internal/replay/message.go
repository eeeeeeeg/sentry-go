package replay

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
	Type        string `json:"type"`
	Category    string `json:"category"`
	Length      int    `json:"length"`
	ContentType string `json:"content_type,omitempty"`
}

type ReplayItem struct {
	MessageID      string
	OrganizationID string
	ProjectID      string
	ProjectKeyID   string
	ReplayID       string
	EventID        string
	TraceID        string
	TransactionID  string
	SegmentID      int
	ItemType       string
	Timestamp      time.Time
	ReceivedAt     time.Time
	SDKName        string
	SDKVersion     string
	ContentType    string
	Size           int64
	Metadata       []byte
	Payload        []byte
}
