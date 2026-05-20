package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"sentry-lite/internal/normalize"
)

type EventWriter struct {
	db *sql.DB
}

func NewEventWriter(db *sql.DB) *EventWriter {
	return &EventWriter{db: db}
}

func (w *EventWriter) InsertEvent(ctx context.Context, event normalize.NormalizedEvent) error {
	tags, err := json.Marshal(emptyMapIfNilString(event.Tags))
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	contexts, err := json.Marshal(emptyMapIfNilAny(event.Contexts))
	if err != nil {
		return fmt.Errorf("marshal contexts: %w", err)
	}

	_, err = w.db.ExecContext(ctx, `
INSERT INTO sentry.events
(
    event_id,
    project_id,
    issue_id,
    timestamp,
    received_at,
    platform,
    runtime_name,
    runtime_version,
    sdk_name,
    sdk_version,
    level,
    message,
    exception_type,
    exception_value,
    release,
    environment,
    user_id,
    tags,
    contexts,
    raw_event
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID,
		event.ProjectID,
		nullableString(event.IssueID),
		event.Timestamp,
		event.ReceivedAt,
		event.Platform,
		event.RuntimeName,
		event.RuntimeVersion,
		event.SDKName,
		event.SDKVersion,
		event.Level,
		event.Message,
		event.ExceptionType,
		event.ExceptionValue,
		event.Release,
		event.Environment,
		event.UserID,
		string(tags),
		string(contexts),
		string(event.RawEvent),
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func emptyMapIfNilString(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func emptyMapIfNilAny(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
