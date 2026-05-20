package attachment

import "time"

type RawEnvelopeItemMessage struct {
	MessageID      string       `json:"message_id"`
	ReceivedAt     time.Time    `json:"received_at"`
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
	Filename    string `json:"filename,omitempty"`
	Attachment  string `json:"attachment_type,omitempty"`
}

type EventAttachment struct {
	MessageID      string
	OrganizationID string
	ProjectID      string
	ProjectKeyID   string
	EventID        string
	Filename       string
	ContentType    string
	AttachmentType string
	Size           int64
	SHA1           string
	Content        []byte
	CreatedAt      time.Time
}
