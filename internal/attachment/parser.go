package attachment

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func ParseRawMessage(body []byte) (EventAttachment, error) {
	var raw RawEnvelopeItemMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return EventAttachment{}, fmt.Errorf("decode raw attachment item: %w", err)
	}
	if raw.Item.Type != "attachment" {
		return EventAttachment{}, nil
	}
	filename := strings.TrimSpace(raw.Item.Filename)
	if filename == "" {
		return EventAttachment{}, fmt.Errorf("attachment filename is required")
	}
	contentType := strings.TrimSpace(raw.Item.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	createdAt := raw.ReceivedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	sum := sha1.Sum(raw.Payload)
	return EventAttachment{
		MessageID:      raw.MessageID,
		OrganizationID: raw.OrganizationID,
		ProjectID:      raw.ProjectID,
		ProjectKeyID:   raw.ProjectKeyID,
		EventID:        raw.EventID,
		Filename:       filename,
		ContentType:    contentType,
		AttachmentType: strings.TrimSpace(raw.Item.Attachment),
		Size:           int64(len(raw.Payload)),
		SHA1:           hex.EncodeToString(sum[:]),
		Content:        raw.Payload,
		CreatedAt:      createdAt,
	}, nil
}
