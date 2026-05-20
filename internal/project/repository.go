package project

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProjectKeyNotFound = errors.New("project key not found")
	ErrProjectDisabled    = errors.New("project is disabled")
	ErrProjectKeyDisabled = errors.New("project key is disabled")
)

type Repository struct {
	db *pgxpool.Pool
}

type ProjectKey struct {
	OrganizationID     string
	ProjectID          string
	SentryProjectID    string
	ProjectSlug        string
	ProjectStatus      string
	KeyID              string
	PublicKey          string
	KeyStatus          string
	RateLimitPerMinute int64
	SampleRate         float64
}

type Project struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	SentryProjectID string    `json:"sentry_project_id"`
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	Platform        string    `json:"platform"`
	Status          string    `json:"status"`
	SampleRate      float64   `json:"sample_rate"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Key struct {
	ID                 string    `json:"id"`
	ProjectID          string    `json:"project_id"`
	PublicKey          string    `json:"public_key"`
	Name               string    `json:"name"`
	Status             string    `json:"status"`
	RateLimitPerMinute int64     `json:"rate_limit_per_minute"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindProjectKey(ctx context.Context, projectRef string, publicKey string) (ProjectKey, error) {
	const query = `
SELECT
    p.organization_id::text,
    p.id::text,
    p.sentry_project_id,
    p.slug,
    p.status,
    pk.id::text,
    pk.public_key,
    pk.status,
    pk.rate_limit_per_minute,
    p.sample_rate::float8
FROM projects p
JOIN project_keys pk ON pk.project_id = p.id
WHERE (p.id::text = $1 OR p.slug = $1 OR p.sentry_project_id = $1)
  AND pk.public_key = $2
LIMIT 1`

	var key ProjectKey
	err := r.db.QueryRow(ctx, query, projectRef, publicKey).Scan(
		&key.OrganizationID,
		&key.ProjectID,
		&key.SentryProjectID,
		&key.ProjectSlug,
		&key.ProjectStatus,
		&key.KeyID,
		&key.PublicKey,
		&key.KeyStatus,
		&key.RateLimitPerMinute,
		&key.SampleRate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProjectKey{}, ErrProjectKeyNotFound
	}
	if err != nil {
		return ProjectKey{}, fmt.Errorf("find project key: %w", err)
	}
	if key.ProjectStatus != "active" {
		return ProjectKey{}, ErrProjectDisabled
	}
	if key.KeyStatus != "active" {
		return ProjectKey{}, ErrProjectKeyDisabled
	}
	return key, nil
}

func (r *Repository) ResolveProjectID(ctx context.Context, projectRef string) (string, error) {
	const query = `
SELECT id::text
FROM projects
WHERE id::text = $1 OR slug = $1 OR sentry_project_id = $1
LIMIT 1`

	var projectID string
	err := r.db.QueryRow(ctx, query, projectRef).Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrProjectKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve project id: %w", err)
	}
	return projectID, nil
}

func (r *Repository) ListProjects(ctx context.Context, limit int, offset int) ([]Project, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
SELECT id::text, organization_id::text, sentry_project_id, slug, name, platform, status, sample_rate::float8, created_at, updated_at, count(*) OVER()::int
FROM projects
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	total := 0
	for rows.Next() {
		var item Project
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.SentryProjectID, &item.Slug, &item.Name, &item.Platform, &item.Status, &item.SampleRate, &item.CreatedAt, &item.UpdatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, total, nil
}

func (r *Repository) GetProject(ctx context.Context, projectRef string) (Project, error) {
	var item Project
	err := r.db.QueryRow(ctx, `
SELECT id::text, organization_id::text, sentry_project_id, slug, name, platform, status, sample_rate::float8, created_at, updated_at
FROM projects
WHERE id::text = $1 OR slug = $1 OR sentry_project_id = $1
LIMIT 1`, projectRef).Scan(&item.ID, &item.OrganizationID, &item.SentryProjectID, &item.Slug, &item.Name, &item.Platform, &item.Status, &item.SampleRate, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, pgx.ErrNoRows
	}
	if err != nil {
		return Project{}, fmt.Errorf("get project: %w", err)
	}
	return item, nil
}

func (r *Repository) CreateProject(ctx context.Context, organizationRef string, slug string, name string, platform string, sampleRate float64) (Project, error) {
	if platform == "" {
		platform = "javascript"
	}
	if sampleRate < 0 || sampleRate > 1 {
		sampleRate = 1
	}
	var item Project
	err := r.db.QueryRow(ctx, `
INSERT INTO projects (organization_id, slug, name, platform, sample_rate)
SELECT id, $2, $3, $4, $5
FROM organizations
WHERE id::text = $1 OR slug = $1
RETURNING id::text, organization_id::text, sentry_project_id, slug, name, platform, status, sample_rate::float8, created_at, updated_at`,
		organizationRef, slug, name, platform, sampleRate,
	).Scan(&item.ID, &item.OrganizationID, &item.SentryProjectID, &item.Slug, &item.Name, &item.Platform, &item.Status, &item.SampleRate, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, pgx.ErrNoRows
	}
	if err != nil {
		return Project{}, fmt.Errorf("create project: %w", err)
	}
	return item, nil
}

func (r *Repository) UpdateProject(ctx context.Context, projectRef string, name string, platform string, sampleRate float64) (Project, error) {
	current, err := r.GetProject(ctx, projectRef)
	if err != nil {
		return Project{}, err
	}
	if name == "" {
		name = current.Name
	}
	if platform == "" {
		platform = current.Platform
	}
	if sampleRate < 0 || sampleRate > 1 {
		sampleRate = current.SampleRate
	}

	var item Project
	err = r.db.QueryRow(ctx, `
UPDATE projects
SET name = $2, platform = $3, sample_rate = $4, updated_at = now()
WHERE id::text = $1
RETURNING id::text, organization_id::text, sentry_project_id, slug, name, platform, status, sample_rate::float8, created_at, updated_at`,
		current.ID, name, platform, sampleRate,
	).Scan(&item.ID, &item.OrganizationID, &item.SentryProjectID, &item.Slug, &item.Name, &item.Platform, &item.Status, &item.SampleRate, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Project{}, fmt.Errorf("update project: %w", err)
	}
	return item, nil
}

func (r *Repository) UpdateProjectStatus(ctx context.Context, projectRef string, status string) (Project, error) {
	if status != "active" && status != "disabled" {
		return Project{}, fmt.Errorf("invalid project status %q", status)
	}
	var item Project
	err := r.db.QueryRow(ctx, `
UPDATE projects
SET status = $2, updated_at = now()
WHERE id::text = $1 OR slug = $1 OR sentry_project_id = $1
RETURNING id::text, organization_id::text, sentry_project_id, slug, name, platform, status, sample_rate::float8, created_at, updated_at`,
		projectRef, status,
	).Scan(&item.ID, &item.OrganizationID, &item.SentryProjectID, &item.Slug, &item.Name, &item.Platform, &item.Status, &item.SampleRate, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, pgx.ErrNoRows
	}
	if err != nil {
		return Project{}, fmt.Errorf("update project status: %w", err)
	}
	return item, nil
}

func (r *Repository) ListProjectKeys(ctx context.Context, projectRef string, limit int, offset int) ([]Key, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
SELECT pk.id::text, pk.project_id::text, pk.public_key, pk.name, pk.status, pk.rate_limit_per_minute, pk.created_at, pk.updated_at, count(*) OVER()::int
FROM project_keys pk
JOIN projects p ON p.id = pk.project_id
WHERE p.id::text = $1 OR p.slug = $1 OR p.sentry_project_id = $1
ORDER BY pk.created_at DESC
LIMIT $2 OFFSET $3`, projectRef, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list project keys: %w", err)
	}
	defer rows.Close()

	var keys []Key
	total := 0
	for rows.Next() {
		var item Key
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.PublicKey, &item.Name, &item.Status, &item.RateLimitPerMinute, &item.CreatedAt, &item.UpdatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan project key: %w", err)
		}
		keys = append(keys, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate project keys: %w", err)
	}
	return keys, total, nil
}

func (r *Repository) CreateProjectKey(ctx context.Context, projectRef string, name string, rateLimit int64) (Key, error) {
	projectID, err := r.ResolveProjectID(ctx, projectRef)
	if err != nil {
		return Key{}, err
	}
	if name == "" {
		name = "Default Key"
	}
	if rateLimit <= 0 {
		rateLimit = 6000
	}

	var item Key
	err = r.db.QueryRow(ctx, `
INSERT INTO project_keys (project_id, public_key, name, rate_limit_per_minute)
VALUES ($1::uuid, $2, $3, $4)
RETURNING id::text, project_id::text, public_key, name, status, rate_limit_per_minute, created_at, updated_at`,
		projectID, newPublicKey(), name, rateLimit,
	).Scan(&item.ID, &item.ProjectID, &item.PublicKey, &item.Name, &item.Status, &item.RateLimitPerMinute, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Key{}, fmt.Errorf("create project key: %w", err)
	}
	return item, nil
}

func (r *Repository) UpdateProjectKey(ctx context.Context, keyID string, name string, rateLimit int64) (Key, error) {
	current, err := r.getProjectKey(ctx, keyID)
	if err != nil {
		return Key{}, err
	}
	if name == "" {
		name = current.Name
	}
	if rateLimit <= 0 {
		rateLimit = current.RateLimitPerMinute
	}
	var item Key
	err = r.db.QueryRow(ctx, `
UPDATE project_keys
SET name = $2, rate_limit_per_minute = $3, updated_at = now()
WHERE id::text = $1
RETURNING id::text, project_id::text, public_key, name, status, rate_limit_per_minute, created_at, updated_at`,
		keyID, name, rateLimit,
	).Scan(&item.ID, &item.ProjectID, &item.PublicKey, &item.Name, &item.Status, &item.RateLimitPerMinute, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Key{}, fmt.Errorf("update project key: %w", err)
	}
	return item, nil
}

func (r *Repository) UpdateProjectKeyStatus(ctx context.Context, keyID string, status string) (Key, error) {
	if status != "active" && status != "disabled" {
		return Key{}, fmt.Errorf("invalid project key status %q", status)
	}
	var item Key
	err := r.db.QueryRow(ctx, `
UPDATE project_keys
SET status = $2, updated_at = now()
WHERE id::text = $1
RETURNING id::text, project_id::text, public_key, name, status, rate_limit_per_minute, created_at, updated_at`,
		keyID, status,
	).Scan(&item.ID, &item.ProjectID, &item.PublicKey, &item.Name, &item.Status, &item.RateLimitPerMinute, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Key{}, pgx.ErrNoRows
	}
	if err != nil {
		return Key{}, fmt.Errorf("update project key status: %w", err)
	}
	return item, nil
}

func (r *Repository) getProjectKey(ctx context.Context, keyID string) (Key, error) {
	var item Key
	err := r.db.QueryRow(ctx, `
SELECT id::text, project_id::text, public_key, name, status, rate_limit_per_minute, created_at, updated_at
FROM project_keys
WHERE id::text = $1`, keyID).Scan(&item.ID, &item.ProjectID, &item.PublicKey, &item.Name, &item.Status, &item.RateLimitPerMinute, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Key{}, pgx.ErrNoRows
	}
	if err != nil {
		return Key{}, fmt.Errorf("get project key: %w", err)
	}
	return item, nil
}

func newPublicKey() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("key-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes[:])
}
