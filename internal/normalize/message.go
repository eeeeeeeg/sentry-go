package normalize

import (
	"encoding/json"
	"time"
)

type RawEventMessage struct {
	MessageID      string                 `json:"message_id"`
	ReceivedAt     time.Time              `json:"received_at"`
	RemoteIP       string                 `json:"remote_ip"`
	UserAgent      string                 `json:"user_agent"`
	SDKName        string                 `json:"sdk_name,omitempty"`
	SDKVersion     string                 `json:"sdk_version,omitempty"`
	OrganizationID string                 `json:"organization_id"`
	ProjectID      string                 `json:"project_id"`
	ProjectRef     string                 `json:"project_ref"`
	ProjectKeyID   string                 `json:"project_key_id"`
	PublicKey      string                 `json:"public_key"`
	EnvelopeItems  []EnvelopeItemMetadata `json:"envelope_items,omitempty"`
	Payload        json.RawMessage        `json:"payload"`
}

type EnvelopeItemMetadata struct {
	Type        string `json:"type"`
	Length      int    `json:"length"`
	ContentType string `json:"content_type,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Attachment  string `json:"attachment_type,omitempty"`
}

type NormalizedEvent struct {
	EventID        string            `json:"event_id"`
	IssueID        string            `json:"issue_id,omitempty"`
	Fingerprint    string            `json:"fingerprint,omitempty"`
	OrganizationID string            `json:"organization_id"`
	ProjectID      string            `json:"project_id"`
	ProjectKeyID   string            `json:"project_key_id"`
	Timestamp      time.Time         `json:"timestamp"`
	ReceivedAt     time.Time         `json:"received_at"`
	Platform       string            `json:"platform"`
	RuntimeName    string            `json:"runtime_name"`
	RuntimeVersion string            `json:"runtime_version"`
	SDKName        string            `json:"sdk_name"`
	SDKVersion     string            `json:"sdk_version"`
	Level          string            `json:"level"`
	Message        string            `json:"message"`
	ExceptionType  string            `json:"exception_type"`
	ExceptionValue string            `json:"exception_value"`
	Release        string            `json:"release"`
	Environment    string            `json:"environment"`
	UserID         string            `json:"user_id"`
	Tags           map[string]string `json:"tags,omitempty"`
	Contexts       map[string]any    `json:"contexts,omitempty"`
	RawEvent       json.RawMessage   `json:"raw_event"`
}
