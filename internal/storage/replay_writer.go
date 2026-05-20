package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"sentry-lite/internal/replay"
)

type ReplayWriter struct {
	db *pgxpool.Pool
}

func NewReplayWriter(db *pgxpool.Pool) *ReplayWriter {
	return &ReplayWriter{db: db}
}

func (w *ReplayWriter) InsertReplayItem(ctx context.Context, item replay.ReplayItem) error {
	_, err := w.db.Exec(ctx, `
INSERT INTO replay_items
(
    organization_id,
    project_id,
    project_key_id,
    replay_id,
    event_id,
    trace_id,
    transaction_id,
    segment_id,
    item_type,
    timestamp,
    received_at,
    sdk_name,
    sdk_version,
    content_type,
    size_bytes,
    metadata,
    payload,
    message_id
)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
ON CONFLICT (message_id) DO NOTHING`,
		item.OrganizationID,
		item.ProjectID,
		item.ProjectKeyID,
		item.ReplayID,
		item.EventID,
		item.TraceID,
		item.TransactionID,
		item.SegmentID,
		item.ItemType,
		item.Timestamp,
		item.ReceivedAt,
		item.SDKName,
		item.SDKVersion,
		item.ContentType,
		item.Size,
		item.Metadata,
		item.Payload,
		item.MessageID,
	)
	if err != nil {
		return fmt.Errorf("insert replay item: %w", err)
	}
	return nil
}
