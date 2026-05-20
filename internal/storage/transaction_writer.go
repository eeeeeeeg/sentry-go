package storage

import (
	"context"
	"database/sql"
	"fmt"

	"sentry-lite/internal/transaction"
)

type TransactionWriter struct {
	db *sql.DB
}

func NewTransactionWriter(db *sql.DB) *TransactionWriter {
	return &TransactionWriter{db: db}
}

func (w *TransactionWriter) InsertTransaction(ctx context.Context, item transaction.TransactionRecord, spans []transaction.SpanRecord) error {
	_, err := w.db.ExecContext(ctx, `
INSERT INTO sentry.transactions
(
    event_id,
    organization_id,
    project_id,
    project_key_id,
    trace_id,
    span_id,
    parent_span_id,
    transaction,
    source,
    operation,
    status,
    start_timestamp,
    end_timestamp,
    duration_ms,
    received_at,
    platform,
    release,
    environment,
    sdk_name,
    sdk_version,
    span_count,
    measurements,
    contexts,
    tags,
    raw_transaction
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.EventID,
		item.OrganizationID,
		item.ProjectID,
		item.ProjectKeyID,
		item.TraceID,
		item.SpanID,
		item.ParentSpanID,
		item.TransactionName,
		item.Source,
		item.Operation,
		item.Status,
		item.StartTimestamp,
		item.EndTimestamp,
		item.DurationMS,
		item.ReceivedAt,
		item.Platform,
		item.Release,
		item.Environment,
		item.SDKName,
		item.SDKVersion,
		item.SpanCount,
		item.Measurements,
		item.Contexts,
		item.Tags,
		string(item.RawTransaction),
	)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	for _, span := range spans {
		if err := w.InsertSpan(ctx, span); err != nil {
			return err
		}
	}
	return nil
}

func (w *TransactionWriter) InsertSpan(ctx context.Context, item transaction.SpanRecord) error {
	_, err := w.db.ExecContext(ctx, `
INSERT INTO sentry.spans
(
    event_id,
    project_id,
    trace_id,
    span_id,
    parent_span_id,
    operation,
    description,
    status,
    start_timestamp,
    end_timestamp,
    duration_ms,
    data
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.EventID,
		item.ProjectID,
		item.TraceID,
		item.SpanID,
		item.ParentSpanID,
		item.Operation,
		item.Description,
		item.Status,
		item.StartTimestamp,
		item.EndTimestamp,
		item.DurationMS,
		item.Data,
	)
	if err != nil {
		return fmt.Errorf("insert span: %w", err)
	}
	return nil
}
