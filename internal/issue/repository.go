package issue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentry-lite/internal/grouping"
	"sentry-lite/internal/normalize"
)

type Repository struct {
	db *pgxpool.Pool
}

type Issue struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Fingerprint string    `json:"fingerprint"`
	Title       string    `json:"title"`
	Culprit     string    `json:"culprit,omitempty"`
	Level       string    `json:"level"`
	Status      string    `json:"status"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	EventCount  int64     `json:"event_count"`
	UserCount   int64     `json:"user_count"`
	Release     string    `json:"release,omitempty"`
	Environment string    `json:"environment,omitempty"`
}

type ListOptions struct {
	ProjectRef  string
	Status      string
	Level       string
	Environment string
	Release     string
	Since       time.Time
	Until       time.Time
	Limit       int
	Offset      int
}

type StatusChange struct {
	ID        string    `json:"id"`
	IssueID   string    `json:"issue_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type UpsertResult struct {
	IssueID   string
	AlertType string
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertFromEvent(ctx context.Context, event normalize.NormalizedEvent, fingerprint string) (UpsertResult, error) {
	userIncrement := 0
	if event.UserID != "" {
		userIncrement = 1
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return UpsertResult{}, fmt.Errorf("begin issue upsert: %w", err)
	}
	defer tx.Rollback(ctx)

	var issueID string
	var oldStatus string
	err = tx.QueryRow(ctx, `
SELECT id::text, status
FROM issues
WHERE project_id = $1 AND fingerprint = $2
FOR UPDATE`, event.ProjectID, fingerprint).Scan(&issueID, &oldStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
INSERT INTO issues
(
    organization_id, project_id, fingerprint, title, culprit, level, status,
    first_seen, last_seen, event_count, user_count, release, environment
)
VALUES ($1, $2, $3, $4, $5, $6, 'unresolved', $7, $7, 1, $8, $9, $10)
RETURNING id::text`,
			event.OrganizationID,
			event.ProjectID,
			fingerprint,
			grouping.Title(event),
			grouping.Culprit(event),
			event.Level,
			event.Timestamp,
			userIncrement,
			emptyToNil(event.Release),
			emptyToNil(event.Environment),
		).Scan(&issueID)
		if err != nil {
			return UpsertResult{}, fmt.Errorf("insert issue: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return UpsertResult{}, fmt.Errorf("commit issue insert: %w", err)
		}
		return UpsertResult{IssueID: issueID, AlertType: "new_issue"}, nil
	}
	if err != nil {
		return UpsertResult{}, fmt.Errorf("lock issue: %w", err)
	}

	newStatus := oldStatus
	alertType := ""
	if oldStatus == "resolved" {
		newStatus = "unresolved"
		alertType = "regression"
	}
	_, err = tx.Exec(ctx, `
UPDATE issues
SET
    status = $2,
    last_seen = GREATEST(last_seen, $3),
    event_count = event_count + 1,
    user_count = user_count + $4,
    level = $5,
    release = COALESCE($6, release),
    environment = COALESCE($7, environment),
    updated_at = now()
WHERE id::text = $1`,
		issueID,
		newStatus,
		event.Timestamp,
		userIncrement,
		event.Level,
		emptyToNil(event.Release),
		emptyToNil(event.Environment),
	)
	if err != nil {
		return UpsertResult{}, fmt.Errorf("update issue: %w", err)
	}

	if oldStatus != newStatus {
		_, err = tx.Exec(ctx, `
INSERT INTO issue_status_changes (issue_id, old_status, new_status, reason)
VALUES ($1::uuid, $2, $3, $4)`, issueID, oldStatus, newStatus, "regression")
		if err != nil {
			return UpsertResult{}, fmt.Errorf("insert regression status change: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return UpsertResult{}, fmt.Errorf("commit issue update: %w", err)
	}
	return UpsertResult{IssueID: issueID, AlertType: alertType}, nil
}

func (r *Repository) List(ctx context.Context, opts ListOptions) ([]Issue, int, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	status := opts.Status
	if status == "" {
		status = "unresolved"
	}

	total, err := r.Count(ctx, opts)
	if err != nil {
		return nil, 0, err
	}

	const query = `
SELECT
    i.id::text,
    i.project_id::text,
    i.fingerprint,
    i.title,
    COALESCE(i.culprit, ''),
    i.level,
    i.status,
    i.first_seen,
    i.last_seen,
    i.event_count,
    i.user_count,
    COALESCE(i.release, ''),
    COALESCE(i.environment, '')
FROM issues i
JOIN projects p ON p.id = i.project_id
WHERE (p.id::text = $1 OR p.slug = $1 OR p.sentry_project_id = $1)
  AND ($2 = 'all' OR i.status = $2)
  AND ($3 = '' OR i.level = $3)
  AND ($4 = '' OR i.environment = $4)
  AND ($5 = '' OR i.release = $5)
  AND ($6::timestamptz IS NULL OR i.last_seen >= $6)
  AND ($7::timestamptz IS NULL OR i.last_seen <= $7)
ORDER BY i.last_seen DESC
LIMIT $8 OFFSET $9`

	rows, err := r.db.Query(ctx, query, opts.ProjectRef, status, opts.Level, opts.Environment, opts.Release, timeArg(opts.Since), timeArg(opts.Until), limit, opts.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list issues: %w", err)
	}
	defer rows.Close()

	issues := []Issue{}
	for rows.Next() {
		var item Issue
		if err := rows.Scan(
			&item.ID,
			&item.ProjectID,
			&item.Fingerprint,
			&item.Title,
			&item.Culprit,
			&item.Level,
			&item.Status,
			&item.FirstSeen,
			&item.LastSeen,
			&item.EventCount,
			&item.UserCount,
			&item.Release,
			&item.Environment,
		); err != nil {
			return nil, 0, fmt.Errorf("scan issue: %w", err)
		}
		issues = append(issues, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate issues: %w", err)
	}
	return issues, total, nil
}

func (r *Repository) Count(ctx context.Context, opts ListOptions) (int, error) {
	status := opts.Status
	if status == "" {
		status = "unresolved"
	}

	const query = `
SELECT COUNT(*)
FROM issues i
JOIN projects p ON p.id = i.project_id
WHERE (p.id::text = $1 OR p.slug = $1 OR p.sentry_project_id = $1)
  AND ($2 = 'all' OR i.status = $2)
  AND ($3 = '' OR i.level = $3)
  AND ($4 = '' OR i.environment = $4)
  AND ($5 = '' OR i.release = $5)
  AND ($6::timestamptz IS NULL OR i.last_seen >= $6)
  AND ($7::timestamptz IS NULL OR i.last_seen <= $7)`

	var total int
	err := r.db.QueryRow(ctx, query, opts.ProjectRef, status, opts.Level, opts.Environment, opts.Release, timeArg(opts.Since), timeArg(opts.Until)).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count issues: %w", err)
	}
	return total, nil
}

func (r *Repository) Get(ctx context.Context, issueID string) (Issue, error) {
	const query = `
SELECT
    id::text,
    project_id::text,
    fingerprint,
    title,
    COALESCE(culprit, ''),
    level,
    status,
    first_seen,
    last_seen,
    event_count,
    user_count,
    COALESCE(release, ''),
    COALESCE(environment, '')
FROM issues
WHERE id::text = $1`

	var item Issue
	err := r.db.QueryRow(ctx, query, issueID).Scan(
		&item.ID,
		&item.ProjectID,
		&item.Fingerprint,
		&item.Title,
		&item.Culprit,
		&item.Level,
		&item.Status,
		&item.FirstSeen,
		&item.LastSeen,
		&item.EventCount,
		&item.UserCount,
		&item.Release,
		&item.Environment,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Issue{}, pgx.ErrNoRows
	}
	if err != nil {
		return Issue{}, fmt.Errorf("get issue: %w", err)
	}
	return item, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, issueID string, status string, reason string) (Issue, error) {
	if !validStatus(status) {
		return Issue{}, fmt.Errorf("invalid status %q", status)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Issue{}, fmt.Errorf("begin issue status update: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM issues WHERE id::text = $1 FOR UPDATE`, issueID).Scan(&oldStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return Issue{}, pgx.ErrNoRows
	}
	if err != nil {
		return Issue{}, fmt.Errorf("lock issue: %w", err)
	}

	if oldStatus != status {
		_, err = tx.Exec(ctx, `
UPDATE issues
SET status = $2, updated_at = now()
WHERE id::text = $1`, issueID, status)
		if err != nil {
			return Issue{}, fmt.Errorf("update issue status: %w", err)
		}

		_, err = tx.Exec(ctx, `
INSERT INTO issue_status_changes (issue_id, old_status, new_status, reason)
VALUES ($1::uuid, $2, $3, $4)`, issueID, oldStatus, status, emptyToNil(reason))
		if err != nil {
			return Issue{}, fmt.Errorf("insert issue status change: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Issue{}, fmt.Errorf("commit issue status update: %w", err)
	}
	return r.Get(ctx, issueID)
}

func (r *Repository) ListStatusChanges(ctx context.Context, issueID string) ([]StatusChange, error) {
	rows, err := r.db.Query(ctx, `
SELECT
    id::text,
    issue_id::text,
    old_status,
    new_status,
    COALESCE(reason, ''),
    created_at
FROM issue_status_changes
WHERE issue_id::text = $1
ORDER BY created_at DESC
LIMIT 100`, issueID)
	if err != nil {
		return nil, fmt.Errorf("list issue status changes: %w", err)
	}
	defer rows.Close()

	var changes []StatusChange
	for rows.Next() {
		var change StatusChange
		if err := rows.Scan(&change.ID, &change.IssueID, &change.OldStatus, &change.NewStatus, &change.Reason, &change.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan issue status change: %w", err)
		}
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate issue status changes: %w", err)
	}
	return changes, nil
}

func validStatus(status string) bool {
	switch status {
	case "unresolved", "resolved", "ignored":
		return true
	default:
		return false
	}
}

func emptyToNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func timeArg(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
