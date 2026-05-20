package storage

import (
	"context"
	"database/sql"
	"fmt"

	"sentry-lite/internal/outcome"
)

type OutcomeWriter struct {
	db *sql.DB
}

func NewOutcomeWriter(db *sql.DB) *OutcomeWriter {
	return &OutcomeWriter{db: db}
}

func (w *OutcomeWriter) InsertOutcome(ctx context.Context, item outcome.Outcome) error {
	_, err := w.db.ExecContext(ctx, `
INSERT INTO sentry.outcomes
(
    project_id,
    project_key_id,
    event_id,
    timestamp,
    received_at,
    category,
    reason,
    quantity,
    source,
    sdk_name,
    sdk_version
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ProjectID,
		item.ProjectKeyID,
		item.EventID,
		item.Timestamp,
		item.ReceivedAt,
		item.Category,
		item.Reason,
		item.Quantity,
		item.Source,
		item.SDKName,
		item.SDKVersion,
	)
	if err != nil {
		return fmt.Errorf("insert outcome: %w", err)
	}
	return nil
}
