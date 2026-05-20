package storage

import (
	"context"
	"database/sql"
	"fmt"

	"sentry-lite/internal/session"
)

type SessionWriter struct {
	db *sql.DB
}

func NewSessionWriter(db *sql.DB) *SessionWriter {
	return &SessionWriter{db: db}
}

func (w *SessionWriter) InsertSession(ctx context.Context, item session.SessionRecord) error {
	_, err := w.db.ExecContext(ctx, `
INSERT INTO sentry.sessions
(
    project_id,
    project_key_id,
    session_id,
    distinct_id_hash,
    started_at,
    bucket,
    timestamp,
    received_at,
    release,
    environment,
    status,
    init,
    sequence,
    errors,
    duration,
    quantity,
    abnormal_mechanism,
    source,
    sdk_name,
    sdk_version
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ProjectID,
		item.ProjectKeyID,
		item.SessionID,
		item.DistinctID,
		item.StartedAt,
		item.Bucket,
		item.Timestamp,
		item.ReceivedAt,
		item.Release,
		item.Environment,
		item.Status,
		item.Init,
		item.Sequence,
		item.Errors,
		item.Duration,
		item.Quantity,
		item.AbnormalMechanism,
		item.Source,
		item.SDKName,
		item.SDKVersion,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}
