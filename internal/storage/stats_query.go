package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TrendPoint struct {
	Bucket time.Time `json:"bucket"`
	Count  uint64    `json:"count"`
}

type BreakdownItem struct {
	Key   string `json:"key"`
	Count uint64 `json:"count"`
}

type ReleaseSummary struct {
	Release     string    `json:"release"`
	EventCount  uint64    `json:"event_count"`
	IssueCount  uint64    `json:"issue_count"`
	UserCount   uint64    `json:"user_count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Environment string    `json:"environment,omitempty"`
}

type TopIssue struct {
	IssueID string `json:"issue_id"`
	Count   uint64 `json:"count"`
}

type StatsQuery struct {
	ProjectID   string
	Environment string
	Release     string
	Since       time.Time
	Until       time.Time
	Limit       int
}

type StatsQuerier struct {
	db *sql.DB
}

func NewStatsQuerier(db *sql.DB) *StatsQuerier {
	return &StatsQuerier{db: db}
}

func (q *StatsQuerier) Trend(ctx context.Context, query StatsQuery) ([]TrendPoint, error) {
	since, until := normalizeRange(query.Since, query.Until)
	rows, err := q.db.QueryContext(ctx, `
SELECT
    toStartOfHour(timestamp) AS bucket,
    count() AS count
FROM sentry.events
WHERE toString(project_id) = ?
  AND timestamp >= ?
  AND timestamp < ?
  AND (? = '' OR environment = ?)
  AND (? = '' OR release = ?)
GROUP BY bucket
ORDER BY bucket ASC`,
		query.ProjectID,
		since,
		until,
		query.Environment, query.Environment,
		query.Release, query.Release,
	)
	if err != nil {
		return nil, fmt.Errorf("query trend: %w", err)
	}
	defer rows.Close()

	var points []TrendPoint
	for rows.Next() {
		var point TrendPoint
		if err := rows.Scan(&point.Bucket, &point.Count); err != nil {
			return nil, fmt.Errorf("scan trend: %w", err)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trend: %w", err)
	}
	return points, nil
}

func (q *StatsQuerier) LevelBreakdown(ctx context.Context, query StatsQuery) ([]BreakdownItem, error) {
	return q.breakdown(ctx, query, "level")
}

func (q *StatsQuerier) TopReleases(ctx context.Context, query StatsQuery) ([]BreakdownItem, error) {
	return q.breakdown(ctx, query, "release")
}

func (q *StatsQuerier) Releases(ctx context.Context, query StatsQuery) ([]ReleaseSummary, error) {
	since, until := normalizeRange(query.Since, query.Until)
	limit := normalizeLimit(query.Limit)
	rows, err := q.db.QueryContext(ctx, `
SELECT
    release,
    count() AS event_count,
    uniqExactIf(issue_id, issue_id IS NOT NULL) AS issue_count,
    uniqExactIf(user_id, user_id != '') AS user_count,
    min(timestamp) AS first_seen,
    max(timestamp) AS last_seen,
    argMax(environment, timestamp) AS latest_environment
FROM sentry.events
WHERE toString(project_id) = ?
  AND timestamp >= ?
  AND timestamp < ?
  AND release != ''
  AND (? = '' OR environment = ?)
  AND (? = '' OR release = ?)
GROUP BY release
ORDER BY last_seen DESC
LIMIT ?`,
		query.ProjectID,
		since,
		until,
		query.Environment, query.Environment,
		query.Release, query.Release,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query releases: %w", err)
	}
	defer rows.Close()

	items := []ReleaseSummary{}
	for rows.Next() {
		var item ReleaseSummary
		if err := rows.Scan(
			&item.Release,
			&item.EventCount,
			&item.IssueCount,
			&item.UserCount,
			&item.FirstSeen,
			&item.LastSeen,
			&item.Environment,
		); err != nil {
			return nil, fmt.Errorf("scan release: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate releases: %w", err)
	}
	return items, nil
}

func (q *StatsQuerier) TopIssues(ctx context.Context, query StatsQuery) ([]TopIssue, error) {
	since, until := normalizeRange(query.Since, query.Until)
	limit := normalizeLimit(query.Limit)
	rows, err := q.db.QueryContext(ctx, `
SELECT
    ifNull(toString(issue_id), '') AS issue_id,
    count() AS count
FROM sentry.events
WHERE toString(project_id) = ?
  AND timestamp >= ?
  AND timestamp < ?
  AND issue_id IS NOT NULL
  AND (? = '' OR environment = ?)
  AND (? = '' OR release = ?)
GROUP BY issue_id
ORDER BY count DESC
LIMIT ?`,
		query.ProjectID,
		since,
		until,
		query.Environment, query.Environment,
		query.Release, query.Release,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query top issues: %w", err)
	}
	defer rows.Close()

	var items []TopIssue
	for rows.Next() {
		var item TopIssue
		if err := rows.Scan(&item.IssueID, &item.Count); err != nil {
			return nil, fmt.Errorf("scan top issue: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top issues: %w", err)
	}
	return items, nil
}

func (q *StatsQuerier) breakdown(ctx context.Context, query StatsQuery, field string) ([]BreakdownItem, error) {
	since, until := normalizeRange(query.Since, query.Until)
	limit := normalizeLimit(query.Limit)
	sqlQuery := fmt.Sprintf(`
SELECT
    %s AS key,
    count() AS count
FROM sentry.events
WHERE toString(project_id) = ?
  AND timestamp >= ?
  AND timestamp < ?
  AND %s != ''
  AND (? = '' OR environment = ?)
  AND (? = '' OR release = ?)
GROUP BY key
ORDER BY count DESC
LIMIT ?`, field, field)

	rows, err := q.db.QueryContext(ctx, sqlQuery,
		query.ProjectID,
		since,
		until,
		query.Environment, query.Environment,
		query.Release, query.Release,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query breakdown %s: %w", field, err)
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.Key, &item.Count); err != nil {
			return nil, fmt.Errorf("scan breakdown %s: %w", field, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate breakdown %s: %w", field, err)
	}
	return items, nil
}

func normalizeRange(since time.Time, until time.Time) (time.Time, time.Time) {
	if until.IsZero() {
		until = time.Now().UTC()
	}
	if since.IsZero() {
		since = until.Add(-24 * time.Hour)
	}
	return since.UTC(), until.UTC()
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 10
	}
	return limit
}
