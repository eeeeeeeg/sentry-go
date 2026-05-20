package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"sentry-lite/internal/attachment"
)

type AttachmentWriter struct {
	db *pgxpool.Pool
}

func NewAttachmentWriter(db *pgxpool.Pool) *AttachmentWriter {
	return &AttachmentWriter{db: db}
}

func (w *AttachmentWriter) InsertAttachment(ctx context.Context, item attachment.EventAttachment) error {
	_, err := w.db.Exec(ctx, `
INSERT INTO event_attachments
(
    organization_id,
    project_id,
    project_key_id,
    event_id,
    message_id,
    filename,
    content_type,
    attachment_type,
    sha1,
    size_bytes,
    content,
    created_at
)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (message_id) DO NOTHING`,
		item.OrganizationID,
		item.ProjectID,
		item.ProjectKeyID,
		item.EventID,
		item.MessageID,
		item.Filename,
		item.ContentType,
		item.AttachmentType,
		item.SHA1,
		item.Size,
		item.Content,
		item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}
	return nil
}
