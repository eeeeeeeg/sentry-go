package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Transaction struct {
	EventID         string    `json:"event_id"`
	ProjectID       string    `json:"project_id"`
	TraceID         string    `json:"trace_id"`
	SpanID          string    `json:"span_id"`
	ParentSpanID    string    `json:"parent_span_id,omitempty"`
	TransactionName string    `json:"transaction"`
	Source          string    `json:"source"`
	Operation       string    `json:"operation"`
	Status          string    `json:"status"`
	StartTimestamp  time.Time `json:"start_timestamp"`
	EndTimestamp    time.Time `json:"end_timestamp"`
	DurationMS      float64   `json:"duration_ms"`
	ReceivedAt      time.Time `json:"received_at"`
	Platform        string    `json:"platform"`
	Release         string    `json:"release,omitempty"`
	Environment     string    `json:"environment,omitempty"`
	SDKName         string    `json:"sdk_name,omitempty"`
	SDKVersion      string    `json:"sdk_version,omitempty"`
	SpanCount       uint64    `json:"span_count"`
	Measurements    string    `json:"measurements"`
	Contexts        string    `json:"contexts"`
	Tags            string    `json:"tags"`
	RawTransaction  string    `json:"raw_transaction,omitempty"`
}

type Span struct {
	EventID        string    `json:"event_id"`
	ProjectID      string    `json:"project_id"`
	TraceID        string    `json:"trace_id"`
	SpanID         string    `json:"span_id"`
	ParentSpanID   string    `json:"parent_span_id,omitempty"`
	Operation      string    `json:"operation"`
	Description    string    `json:"description,omitempty"`
	Status         string    `json:"status,omitempty"`
	StartTimestamp time.Time `json:"start_timestamp"`
	EndTimestamp   time.Time `json:"end_timestamp"`
	DurationMS     float64   `json:"duration_ms"`
	Data           string    `json:"data"`
}

type TransactionQuery struct {
	ProjectID   string
	Operation   string
	Environment string
	Release     string
	Query       string
	Since       time.Time
	Until       time.Time
	Limit       int
	Offset      int
}

type TransactionQuerier struct {
	db *sql.DB
}

func NewTransactionQuerier(db *sql.DB) *TransactionQuerier {
	return &TransactionQuerier{db: db}
}

func (q *TransactionQuerier) List(ctx context.Context, query TransactionQuery) ([]Transaction, int, error) {
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	whereSQL, args := transactionWhere(query)
	countArgs := append([]any{}, args...)
	var totalValue uint64
	if err := q.db.QueryRowContext(ctx, "SELECT count() FROM sentry.transactions "+whereSQL, countArgs...).Scan(&totalValue); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	listArgs := append(args, limit, query.Offset)
	rows, err := q.db.QueryContext(ctx, `
SELECT
    toString(event_id),
    toString(project_id),
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
    tags
FROM sentry.transactions
`+whereSQL+`
ORDER BY start_timestamp DESC
LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	items := []Transaction{}
	for rows.Next() {
		var item Transaction
		if err := scanTransaction(rows, &item, false); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate transactions: %w", err)
	}
	return items, int(totalValue), nil
}

func (q *TransactionQuerier) Get(ctx context.Context, eventID string) (Transaction, error) {
	eventID = canonicalQueryUUID(eventID)
	var item Transaction
	err := q.db.QueryRowContext(ctx, `
SELECT
    toString(event_id),
    toString(project_id),
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
FROM sentry.transactions
WHERE event_id = toUUID(?)
LIMIT 1`, eventID).Scan(
		&item.EventID,
		&item.ProjectID,
		&item.TraceID,
		&item.SpanID,
		&item.ParentSpanID,
		&item.TransactionName,
		&item.Source,
		&item.Operation,
		&item.Status,
		&item.StartTimestamp,
		&item.EndTimestamp,
		&item.DurationMS,
		&item.ReceivedAt,
		&item.Platform,
		&item.Release,
		&item.Environment,
		&item.SDKName,
		&item.SDKVersion,
		&item.SpanCount,
		&item.Measurements,
		&item.Contexts,
		&item.Tags,
		&item.RawTransaction,
	)
	if err != nil {
		return Transaction{}, fmt.Errorf("get transaction: %w", err)
	}
	return item, nil
}

func (q *TransactionQuerier) ListSpans(ctx context.Context, eventID string) ([]Span, error) {
	eventID = canonicalQueryUUID(eventID)
	rows, err := q.db.QueryContext(ctx, `
SELECT
    toString(event_id),
    toString(project_id),
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
FROM sentry.spans
WHERE event_id = toUUID(?)
ORDER BY start_timestamp ASC`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list spans: %w", err)
	}
	defer rows.Close()

	items := []Span{}
	for rows.Next() {
		var item Span
		if err := rows.Scan(
			&item.EventID,
			&item.ProjectID,
			&item.TraceID,
			&item.SpanID,
			&item.ParentSpanID,
			&item.Operation,
			&item.Description,
			&item.Status,
			&item.StartTimestamp,
			&item.EndTimestamp,
			&item.DurationMS,
			&item.Data,
		); err != nil {
			return nil, fmt.Errorf("scan span: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spans: %w", err)
	}
	return items, nil
}

type transactionScanner interface {
	Scan(dest ...any) error
}

func scanTransaction(row transactionScanner, item *Transaction, includeRaw bool) error {
	dest := []any{
		&item.EventID,
		&item.ProjectID,
		&item.TraceID,
		&item.SpanID,
		&item.ParentSpanID,
		&item.TransactionName,
		&item.Source,
		&item.Operation,
		&item.Status,
		&item.StartTimestamp,
		&item.EndTimestamp,
		&item.DurationMS,
		&item.ReceivedAt,
		&item.Platform,
		&item.Release,
		&item.Environment,
		&item.SDKName,
		&item.SDKVersion,
		&item.SpanCount,
		&item.Measurements,
		&item.Contexts,
		&item.Tags,
	}
	if includeRaw {
		dest = append(dest, &item.RawTransaction)
	}
	if err := row.Scan(dest...); err != nil {
		return fmt.Errorf("scan transaction: %w", err)
	}
	return nil
}

func transactionWhere(query TransactionQuery) (string, []any) {
	conditions := []string{"toString(project_id) = ?"}
	args := []any{query.ProjectID}

	if query.Operation != "" {
		conditions = append(conditions, "operation = ?")
		args = append(args, query.Operation)
	}
	if query.Environment != "" {
		conditions = append(conditions, "environment = ?")
		args = append(args, query.Environment)
	}
	if query.Release != "" {
		conditions = append(conditions, "release = ?")
		args = append(args, query.Release)
	}
	if query.Query != "" {
		conditions = append(conditions, "positionCaseInsensitive(transaction, ?) > 0")
		args = append(args, query.Query)
	}
	if !query.Since.IsZero() {
		conditions = append(conditions, "start_timestamp >= ?")
		args = append(args, query.Since)
	}
	if !query.Until.IsZero() {
		conditions = append(conditions, "start_timestamp <= ?")
		args = append(args, query.Until)
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

func canonicalQueryUUID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) != 32 {
		return value
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
			return value
		}
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", value[0:8], value[8:12], value[12:16], value[16:20], value[20:32])
}
