package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Event struct {
	EventID        string    `json:"event_id"`
	ProjectID      string    `json:"project_id"`
	IssueID        string    `json:"issue_id,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	ReceivedAt     time.Time `json:"received_at"`
	Platform       string    `json:"platform"`
	Level          string    `json:"level"`
	Message        string    `json:"message"`
	ExceptionType  string    `json:"exception_type"`
	ExceptionValue string    `json:"exception_value"`
	Release        string    `json:"release,omitempty"`
	Environment    string    `json:"environment,omitempty"`
	UserID         string    `json:"user_id,omitempty"`
	Tags           string    `json:"tags"`
	Contexts       string    `json:"contexts"`
	RawEvent       string    `json:"raw_event,omitempty"`
}

type EventQuery struct {
	ProjectID   string
	IssueID     string
	Level       string
	Environment string
	Release     string
	Since       time.Time
	Until       time.Time
	Limit       int
	Offset      int
}

type EventQuerier struct {
	db *sql.DB
}

func NewEventQuerier(db *sql.DB) *EventQuerier {
	return &EventQuerier{db: db}
}

func (q *EventQuerier) List(ctx context.Context, query EventQuery) ([]Event, int, error) {
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	whereSQL, args := eventWhere(query)
	countArgs := append([]any{}, args...)
	var totalValue uint64
	if err := q.db.QueryRowContext(ctx, "SELECT count() FROM sentry.events "+whereSQL, countArgs...).Scan(&totalValue); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	listArgs := append(args, limit, query.Offset)
	rows, err := q.db.QueryContext(ctx, `
SELECT
    toString(event_id),
    toString(project_id),
    ifNull(toString(issue_id), ''),
    timestamp,
    received_at,
    platform,
    level,
    message,
    exception_type,
    exception_value,
    release,
    environment,
    user_id,
    tags,
    contexts
FROM sentry.events
`+whereSQL+`
ORDER BY timestamp DESC
LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.EventID,
			&event.ProjectID,
			&event.IssueID,
			&event.Timestamp,
			&event.ReceivedAt,
			&event.Platform,
			&event.Level,
			&event.Message,
			&event.ExceptionType,
			&event.ExceptionValue,
			&event.Release,
			&event.Environment,
			&event.UserID,
			&event.Tags,
			&event.Contexts,
		); err != nil {
			return nil, 0, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate events: %w", err)
	}
	return events, int(totalValue), nil
}

func (q *EventQuerier) Get(ctx context.Context, eventID string) (Event, error) {
	var event Event
	err := q.db.QueryRowContext(ctx, `
SELECT
    toString(event_id),
    toString(project_id),
    ifNull(toString(issue_id), ''),
    timestamp,
    received_at,
    platform,
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
FROM sentry.events
WHERE event_id = toUUID(?)
LIMIT 1`, eventID).Scan(
		&event.EventID,
		&event.ProjectID,
		&event.IssueID,
		&event.Timestamp,
		&event.ReceivedAt,
		&event.Platform,
		&event.Level,
		&event.Message,
		&event.ExceptionType,
		&event.ExceptionValue,
		&event.Release,
		&event.Environment,
		&event.UserID,
		&event.Tags,
		&event.Contexts,
		&event.RawEvent,
	)
	if err != nil {
		return Event{}, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}

func eventWhere(query EventQuery) (string, []any) {
	conditions := []string{"toString(project_id) = ?"}
	args := []any{query.ProjectID}

	if query.IssueID != "" {
		conditions = append(conditions, "ifNull(toString(issue_id), '') = ?")
		args = append(args, query.IssueID)
	}
	if query.Level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, query.Level)
	}
	if query.Environment != "" {
		conditions = append(conditions, "environment = ?")
		args = append(args, query.Environment)
	}
	if query.Release != "" {
		conditions = append(conditions, "release = ?")
		args = append(args, query.Release)
	}
	if !query.Since.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, query.Since)
	}
	if !query.Until.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, query.Until)
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}
