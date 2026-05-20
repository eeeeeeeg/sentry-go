package alert

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Rule struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	EventType       string `json:"event_type"`
	Channel         string `json:"channel"`
	WebhookURL      string `json:"webhook_url,omitempty"`
	MinLevel        string `json:"min_level"`
	ThresholdCount  int    `json:"threshold_count"`
	WindowSeconds   int    `json:"window_seconds"`
	CooldownSeconds int    `json:"cooldown_seconds"`
	Status          string `json:"status"`
}

type Delivery struct {
	ID          string    `json:"id"`
	AlertID     string    `json:"alert_id,omitempty"`
	ProjectID   string    `json:"project_id"`
	IssueID     string    `json:"issue_id"`
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	Channel     string    `json:"channel"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	DeliveredAt time.Time `json:"delivered_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateWebhookRule(ctx context.Context, projectID string, name string, eventType string, webhookURL string, minLevel string, thresholdCount int, windowSeconds int, cooldownSeconds int) (Rule, error) {
	if eventType == "" {
		eventType = "new_issue"
	}
	if minLevel == "" {
		minLevel = "error"
	}
	if thresholdCount <= 0 {
		thresholdCount = 1
	}
	if windowSeconds <= 0 {
		windowSeconds = 300
	}
	if cooldownSeconds <= 0 {
		cooldownSeconds = 300
	}

	const query = `
INSERT INTO alerts
(
    project_id,
    name,
    event_type,
    channel,
    webhook_url,
    min_level,
    threshold_count,
    window_seconds,
    cooldown_seconds
)
VALUES ($1::uuid, $2, $3, 'webhook', $4, $5, $6, $7, $8)
RETURNING id::text, project_id::text, name, event_type, channel, COALESCE(webhook_url, ''), min_level, threshold_count, window_seconds, cooldown_seconds, status`

	var rule Rule
	err := r.db.QueryRow(ctx, query, projectID, name, eventType, webhookURL, minLevel, thresholdCount, windowSeconds, cooldownSeconds).Scan(
		&rule.ID,
		&rule.ProjectID,
		&rule.Name,
		&rule.EventType,
		&rule.Channel,
		&rule.WebhookURL,
		&rule.MinLevel,
		&rule.ThresholdCount,
		&rule.WindowSeconds,
		&rule.CooldownSeconds,
		&rule.Status,
	)
	if err != nil {
		return Rule{}, fmt.Errorf("create webhook alert rule: %w", err)
	}
	return rule, nil
}

func (r *Repository) ListProjectRules(ctx context.Context, projectID string) ([]Rule, error) {
	rows, err := r.db.Query(ctx, `
SELECT
    id::text,
    project_id::text,
    name,
    event_type,
    channel,
    COALESCE(webhook_url, ''),
    min_level,
    threshold_count,
    window_seconds,
    cooldown_seconds,
    status
FROM alerts
WHERE project_id::text = $1
ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project alerts: %w", err)
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *Repository) ListActiveRules(ctx context.Context, projectID string, eventType string) ([]Rule, error) {
	rows, err := r.db.Query(ctx, `
SELECT
    id::text,
    project_id::text,
    name,
    event_type,
    channel,
    COALESCE(webhook_url, ''),
    min_level,
    threshold_count,
    window_seconds,
    cooldown_seconds,
    status
FROM alerts
WHERE project_id::text = $1
  AND event_type = $2
  AND status = 'active'
ORDER BY created_at ASC`, projectID, eventType)
	if err != nil {
		return nil, fmt.Errorf("list active alerts: %w", err)
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *Repository) GetRule(ctx context.Context, ruleID string) (Rule, error) {
	const query = `
SELECT
    id::text,
    project_id::text,
    name,
    event_type,
    channel,
    COALESCE(webhook_url, ''),
    min_level,
    threshold_count,
    window_seconds,
    cooldown_seconds,
    status
FROM alerts
WHERE id::text = $1`

	var rule Rule
	err := r.db.QueryRow(ctx, query, ruleID).Scan(
		&rule.ID,
		&rule.ProjectID,
		&rule.Name,
		&rule.EventType,
		&rule.Channel,
		&rule.WebhookURL,
		&rule.MinLevel,
		&rule.ThresholdCount,
		&rule.WindowSeconds,
		&rule.CooldownSeconds,
		&rule.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, pgx.ErrNoRows
	}
	if err != nil {
		return Rule{}, fmt.Errorf("get alert rule: %w", err)
	}
	return rule, nil
}

type ruleRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanRules(rows ruleRows) ([]Rule, error) {
	var rules []Rule
	for rows.Next() {
		var rule Rule
		if err := rows.Scan(
			&rule.ID,
			&rule.ProjectID,
			&rule.Name,
			&rule.EventType,
			&rule.Channel,
			&rule.WebhookURL,
			&rule.MinLevel,
			&rule.ThresholdCount,
			&rule.WindowSeconds,
			&rule.CooldownSeconds,
			&rule.Status,
		); err != nil {
			return nil, fmt.Errorf("scan alert rule: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert rules: %w", err)
	}
	return rules, nil
}

func (r *Repository) UpdateRuleStatus(ctx context.Context, ruleID string, status string) (Rule, error) {
	if status != "active" && status != "disabled" {
		return Rule{}, fmt.Errorf("invalid alert status %q", status)
	}

	const query = `
UPDATE alerts
SET status = $2, updated_at = now()
WHERE id::text = $1
RETURNING id::text, project_id::text, name, event_type, channel, COALESCE(webhook_url, ''), min_level, threshold_count, window_seconds, cooldown_seconds, status`

	var rule Rule
	err := r.db.QueryRow(ctx, query, ruleID, status).Scan(
		&rule.ID,
		&rule.ProjectID,
		&rule.Name,
		&rule.EventType,
		&rule.Channel,
		&rule.WebhookURL,
		&rule.MinLevel,
		&rule.ThresholdCount,
		&rule.WindowSeconds,
		&rule.CooldownSeconds,
		&rule.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, pgx.ErrNoRows
	}
	if err != nil {
		return Rule{}, fmt.Errorf("update alert status: %w", err)
	}
	return rule, nil
}

func (r *Repository) ListProjectDeliveries(ctx context.Context, projectID string, status string, limit int, offset int) ([]Delivery, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var total int
	err := r.db.QueryRow(ctx, `
SELECT COUNT(*)
FROM alert_deliveries
WHERE project_id::text = $1
  AND ($2 = '' OR status = $2)`, projectID, status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count alert deliveries: %w", err)
	}

	rows, err := r.db.Query(ctx, `
SELECT
    id::text,
    COALESCE(alert_id::text, ''),
    project_id::text,
    issue_id::text,
    event_id,
    event_type,
    channel,
    status,
    COALESCE(error, ''),
    COALESCE(delivered_at, '0001-01-01T00:00:00Z'::timestamptz),
    created_at
FROM alert_deliveries
WHERE project_id::text = $1
  AND ($2 = '' OR status = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4`, projectID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list alert deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := []Delivery{}
	for rows.Next() {
		var delivery Delivery
		if err := rows.Scan(
			&delivery.ID,
			&delivery.AlertID,
			&delivery.ProjectID,
			&delivery.IssueID,
			&delivery.EventID,
			&delivery.EventType,
			&delivery.Channel,
			&delivery.Status,
			&delivery.Error,
			&delivery.DeliveredAt,
			&delivery.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan alert delivery: %w", err)
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate alert deliveries: %w", err)
	}
	return deliveries, total, nil
}

func (r *Repository) RecordDelivery(ctx context.Context, ruleID string, event Event, status string, channel string, cause string) error {
	var deliveredAt any
	if status == "sent" {
		deliveredAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx, `
INSERT INTO alert_deliveries
(
    alert_id,
    project_id,
    issue_id,
    event_id,
    event_type,
    channel,
    status,
    error,
    delivered_at
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9)`,
		ruleID,
		event.ProjectID,
		event.IssueID,
		event.EventID,
		event.Type,
		channel,
		status,
		emptyToNil(cause),
		deliveredAt,
	)
	if err != nil {
		return fmt.Errorf("record alert delivery: %w", err)
	}
	return nil
}

func emptyToNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
