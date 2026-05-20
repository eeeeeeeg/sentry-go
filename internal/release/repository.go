package release

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrProjectNotFound      = errors.New("project not found")
	ErrReleaseNotFound      = errors.New("release not found")
	ErrReleaseFileNotFound  = errors.New("release file not found")
)

type Repository struct {
	db *pgxpool.Pool
}

type Release struct {
	ID           string           `json:"id"`
	Authors      []any            `json:"authors"`
	CommitCount  int              `json:"commitCount"`
	Data         map[string]any   `json:"data"`
	DateCreated  time.Time        `json:"dateCreated"`
	DateReleased any              `json:"dateReleased"`
	DeployCount  int              `json:"deployCount"`
	FirstEvent   any              `json:"firstEvent"`
	LastCommit   any              `json:"lastCommit"`
	LastDeploy   any              `json:"lastDeploy"`
	LastEvent    any              `json:"lastEvent"`
	NewGroups    int              `json:"newGroups"`
	Owner        any              `json:"owner"`
	Projects     []ReleaseProject `json:"projects"`
	Ref          any              `json:"ref"`
	ShortVersion string           `json:"shortVersion"`
	URL          any              `json:"url"`
	Version      string           `json:"version"`
}

type Deploy struct {
	ID           string    `json:"id"`
	Environment  string    `json:"environment"`
	Name         any       `json:"name"`
	URL          any       `json:"url"`
	DateStarted  any       `json:"dateStarted"`
	DateFinished time.Time `json:"dateFinished"`
	DateCreated  time.Time `json:"dateCreated"`
}

type File struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Dist        any               `json:"dist"`
	Headers     map[string]string `json:"headers"`
	SHA1        string            `json:"sha1"`
	Size        int64             `json:"size"`
	DateCreated time.Time         `json:"dateCreated"`
}

type ReleaseProject struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CreateReleaseInput struct {
	Version      string
	Projects     []string
	Ref          string
	URL          string
	DateReleased string
}

type UploadFileInput struct {
	OrganizationRef string
	ProjectRef      string
	Version         string
	Name            string
	Dist            string
	Headers         map[string]string
	Content         []byte
}

type CreateDeployInput struct {
	Environment  string
	Name         string
	URL          string
	DateStarted  string
	DateFinished string
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOrUpdate(ctx context.Context, organizationRef string, input CreateReleaseInput) (Release, error) {
	orgID, err := r.resolveOrganization(ctx, organizationRef)
	if err != nil {
		return Release{}, err
	}

	var item Release
	var organizationID string
	var ref string
	var releaseURL string
	var dateReleased string
	err = r.db.QueryRow(ctx, `
INSERT INTO releases (organization_id, version, ref, url, date_released, updated_at)
VALUES ($1::uuid, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, '')::timestamptz, now())
ON CONFLICT (organization_id, version)
DO UPDATE SET ref = COALESCE(NULLIF(EXCLUDED.ref, ''), releases.ref),
              url = COALESCE(NULLIF(EXCLUDED.url, ''), releases.url),
              date_released = COALESCE(EXCLUDED.date_released, releases.date_released),
              updated_at = now()
RETURNING id::text, organization_id::text, version, COALESCE(ref, ''), COALESCE(url, ''), COALESCE(date_released::text, ''), created_at`,
		orgID, input.Version, input.Ref, input.URL, input.DateReleased,
	).Scan(&item.ID, &organizationID, &item.Version, &ref, &releaseURL, &dateReleased, &item.DateCreated)
	if err != nil {
		return Release{}, fmt.Errorf("upsert release: %w", err)
	}

	for _, projectRef := range input.Projects {
		projectID, err := r.resolveProject(ctx, orgID, projectRef)
		if err != nil {
			return Release{}, err
		}
		if _, err := r.db.Exec(ctx, `
INSERT INTO release_projects (release_id, project_id)
VALUES ($1::uuid, $2::uuid)
ON CONFLICT DO NOTHING`, item.ID, projectID); err != nil {
			return Release{}, fmt.Errorf("link release project: %w", err)
		}
	}

	return r.withReleaseResponseDefaults(ctx, item, ref, releaseURL, dateReleased)
}

func (r *Repository) List(ctx context.Context, organizationRef string, query string, limit int) ([]Release, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	orgID, err := r.resolveOrganization(ctx, organizationRef)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
SELECT id::text, organization_id::text, version, COALESCE(ref, ''), COALESCE(url, ''), COALESCE(date_released::text, ''), created_at
FROM releases
WHERE organization_id = $1::uuid
  AND ($2 = '' OR version ILIKE $2 || '%')
ORDER BY created_at DESC
LIMIT $3`, orgID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}
	defer rows.Close()

	var items []Release
	for rows.Next() {
		var item Release
		var organizationID string
		var ref string
		var releaseURL string
		var dateReleased string
		if err := rows.Scan(&item.ID, &organizationID, &item.Version, &ref, &releaseURL, &dateReleased, &item.DateCreated); err != nil {
			return nil, fmt.Errorf("scan release: %w", err)
		}
		item, err = r.withReleaseResponseDefaults(ctx, item, ref, releaseURL, dateReleased)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate releases: %w", err)
	}
	return items, nil
}

func (r *Repository) Get(ctx context.Context, organizationRef string, version string) (Release, error) {
	orgID, err := r.resolveOrganization(ctx, organizationRef)
	if err != nil {
		return Release{}, err
	}

	var item Release
	var organizationID string
	var ref string
	var releaseURL string
	var dateReleased string
	err = r.db.QueryRow(ctx, `
SELECT id::text, organization_id::text, version, COALESCE(ref, ''), COALESCE(url, ''), COALESCE(date_released::text, ''), created_at
FROM releases
WHERE organization_id = $1::uuid AND version = $2
LIMIT 1`, orgID, version).Scan(&item.ID, &organizationID, &item.Version, &ref, &releaseURL, &dateReleased, &item.DateCreated)
	if errors.Is(err, pgx.ErrNoRows) {
		return Release{}, ErrReleaseNotFound
	}
	if err != nil {
		return Release{}, fmt.Errorf("get release: %w", err)
	}
	return r.withReleaseResponseDefaults(ctx, item, ref, releaseURL, dateReleased)
}

func (r *Repository) Update(ctx context.Context, organizationRef string, version string, input CreateReleaseInput) (Release, error) {
	orgID, err := r.resolveOrganization(ctx, organizationRef)
	if err != nil {
		return Release{}, err
	}

	var item Release
	var organizationID string
	var ref string
	var releaseURL string
	var dateReleased string
	err = r.db.QueryRow(ctx, `
UPDATE releases
SET ref = COALESCE(NULLIF($3, ''), ref),
    url = COALESCE(NULLIF($4, ''), url),
    date_released = COALESCE(NULLIF($5, '')::timestamptz, date_released),
    updated_at = now()
WHERE organization_id = $1::uuid AND version = $2
RETURNING id::text, organization_id::text, version, COALESCE(ref, ''), COALESCE(url, ''), COALESCE(date_released::text, ''), created_at`,
		orgID, version, input.Ref, input.URL, input.DateReleased,
	).Scan(&item.ID, &organizationID, &item.Version, &ref, &releaseURL, &dateReleased, &item.DateCreated)
	if errors.Is(err, pgx.ErrNoRows) {
		return Release{}, ErrReleaseNotFound
	}
	if err != nil {
		return Release{}, fmt.Errorf("update release: %w", err)
	}
	return r.withReleaseResponseDefaults(ctx, item, ref, releaseURL, dateReleased)
}

func (r *Repository) Delete(ctx context.Context, organizationRef string, version string) error {
	orgID, err := r.resolveOrganization(ctx, organizationRef)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(ctx, `
DELETE FROM releases
WHERE organization_id = $1::uuid AND version = $2`, orgID, version)
	if err != nil {
		return fmt.Errorf("delete release: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrReleaseNotFound
	}
	return nil
}

func (r *Repository) UploadFile(ctx context.Context, input UploadFileInput) (File, error) {
	releaseID, err := r.ensureRelease(ctx, input.OrganizationRef, input.ProjectRef, input.Version)
	if err != nil {
		return File{}, err
	}

	projectID := ""
	if input.ProjectRef != "" {
		orgID, err := r.resolveOrganization(ctx, input.OrganizationRef)
		if err != nil {
			return File{}, err
		}
		projectID, err = r.resolveProject(ctx, orgID, input.ProjectRef)
		if err != nil {
			return File{}, err
		}
	}

	headers, err := json.Marshal(input.Headers)
	if err != nil {
		return File{}, fmt.Errorf("marshal headers: %w", err)
	}
	sum := sha1.Sum(input.Content)
	sha := hex.EncodeToString(sum[:])

	var item File
	var dist string
	err = r.db.QueryRow(ctx, `
INSERT INTO release_files (release_id, project_id, name, dist, headers, sha1, size_bytes, content)
VALUES ($1::uuid, NULLIF($2, '')::uuid, $3, NULLIF($4, ''), $5::jsonb, $6, $7, $8)
ON CONFLICT (release_id, (COALESCE(project_id, '00000000-0000-0000-0000-000000000000'::uuid)), name, (COALESCE(dist, '')))
DO UPDATE SET headers = EXCLUDED.headers,
              sha1 = EXCLUDED.sha1,
              size_bytes = EXCLUDED.size_bytes,
              content = EXCLUDED.content,
              created_at = now()
RETURNING id::text, name, COALESCE(dist, ''), headers::text, sha1, size_bytes, created_at`,
		releaseID, projectID, input.Name, input.Dist, string(headers), sha, int64(len(input.Content)), input.Content,
	).Scan(&item.ID, &item.Name, &dist, &headers, &item.SHA1, &item.Size, &item.DateCreated)
	if err != nil {
		return File{}, fmt.Errorf("upsert release file: %w", err)
	}
	item.Dist = nullableStringValue(dist)
	_ = json.Unmarshal(headers, &item.Headers)
	if item.Headers == nil {
		item.Headers = map[string]string{}
	}
	return item, nil
}

func (r *Repository) ListFiles(ctx context.Context, organizationRef string, projectRef string, version string) ([]File, error) {
	releaseID, err := r.releaseID(ctx, organizationRef, version)
	if err != nil {
		return nil, err
	}

	projectFilter := ""
	if projectRef != "" {
		orgID, err := r.resolveOrganization(ctx, organizationRef)
		if err != nil {
			return nil, err
		}
		projectFilter, err = r.resolveProject(ctx, orgID, projectRef)
		if err != nil {
			return nil, err
		}
	}

	rows, err := r.db.Query(ctx, `
SELECT id::text, name, COALESCE(dist, ''), headers::text, sha1, size_bytes, created_at
FROM release_files
WHERE release_id = $1::uuid
  AND ($2 = '' OR project_id = $2::uuid)
ORDER BY created_at DESC`, releaseID, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("list release files: %w", err)
	}
	defer rows.Close()

	var items []File
	for rows.Next() {
		var item File
		var headers []byte
		var dist string
		if err := rows.Scan(&item.ID, &item.Name, &dist, &headers, &item.SHA1, &item.Size, &item.DateCreated); err != nil {
			return nil, fmt.Errorf("scan release file: %w", err)
		}
		item.Dist = nullableStringValue(dist)
		_ = json.Unmarshal(headers, &item.Headers)
		if item.Headers == nil {
			item.Headers = map[string]string{}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate release files: %w", err)
	}
	return items, nil
}

func (r *Repository) GetFile(ctx context.Context, organizationRef string, projectRef string, version string, fileID string) (File, []byte, error) {
	releaseID, err := r.releaseID(ctx, organizationRef, version)
	if err != nil {
		return File{}, nil, err
	}
	projectFilter, err := r.optionalProjectID(ctx, organizationRef, projectRef)
	if err != nil {
		return File{}, nil, err
	}

	var item File
	var headers []byte
	var dist string
	var content []byte
	err = r.db.QueryRow(ctx, `
SELECT id::text, name, COALESCE(dist, ''), headers::text, sha1, size_bytes, created_at, content
FROM release_files
WHERE release_id = $1::uuid
  AND id::text = $2
  AND ($3 = '' OR project_id = $3::uuid)
LIMIT 1`, releaseID, fileID, projectFilter).Scan(&item.ID, &item.Name, &dist, &headers, &item.SHA1, &item.Size, &item.DateCreated, &content)
	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, nil, ErrReleaseFileNotFound
	}
	if err != nil {
		return File{}, nil, fmt.Errorf("get release file: %w", err)
	}
	item.Dist = nullableStringValue(dist)
	_ = json.Unmarshal(headers, &item.Headers)
	if item.Headers == nil {
		item.Headers = map[string]string{}
	}
	return item, content, nil
}

func (r *Repository) UpdateFile(ctx context.Context, organizationRef string, projectRef string, version string, fileID string, name string, dist string) (File, error) {
	releaseID, err := r.releaseID(ctx, organizationRef, version)
	if err != nil {
		return File{}, err
	}
	projectFilter, err := r.optionalProjectID(ctx, organizationRef, projectRef)
	if err != nil {
		return File{}, err
	}

	var item File
	var headers []byte
	var storedDist string
	err = r.db.QueryRow(ctx, `
UPDATE release_files
SET name = COALESCE(NULLIF($4, ''), name),
    dist = CASE WHEN $5 = '' THEN dist ELSE $5 END
WHERE release_id = $1::uuid
  AND id::text = $2
  AND ($3 = '' OR project_id = $3::uuid)
RETURNING id::text, name, COALESCE(dist, ''), headers::text, sha1, size_bytes, created_at`,
		releaseID, fileID, projectFilter, name, dist,
	).Scan(&item.ID, &item.Name, &storedDist, &headers, &item.SHA1, &item.Size, &item.DateCreated)
	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, ErrReleaseFileNotFound
	}
	if err != nil {
		return File{}, fmt.Errorf("update release file: %w", err)
	}
	item.Dist = nullableStringValue(storedDist)
	_ = json.Unmarshal(headers, &item.Headers)
	if item.Headers == nil {
		item.Headers = map[string]string{}
	}
	return item, nil
}

func (r *Repository) DeleteFile(ctx context.Context, organizationRef string, projectRef string, version string, fileID string) error {
	releaseID, err := r.releaseID(ctx, organizationRef, version)
	if err != nil {
		return err
	}
	projectFilter, err := r.optionalProjectID(ctx, organizationRef, projectRef)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(ctx, `
DELETE FROM release_files
WHERE release_id = $1::uuid
  AND id::text = $2
  AND ($3 = '' OR project_id = $3::uuid)`, releaseID, fileID, projectFilter)
	if err != nil {
		return fmt.Errorf("delete release file: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrReleaseFileNotFound
	}
	return nil
}

func (r *Repository) CreateDeploy(ctx context.Context, organizationRef string, version string, input CreateDeployInput) (Deploy, error) {
	releaseID, err := r.releaseID(ctx, organizationRef, version)
	if err != nil {
		return Deploy{}, err
	}

	var item Deploy
	var name string
	var deployURL string
	var dateStarted string
	err = r.db.QueryRow(ctx, `
INSERT INTO release_deploys (release_id, environment, name, url, date_started, date_finished)
VALUES ($1::uuid, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, '')::timestamptz, COALESCE(NULLIF($6, '')::timestamptz, now()))
RETURNING id::text, environment, COALESCE(name, ''), COALESCE(url, ''), COALESCE(date_started::text, ''), date_finished, created_at`,
		releaseID, input.Environment, input.Name, input.URL, input.DateStarted, input.DateFinished,
	).Scan(&item.ID, &item.Environment, &name, &deployURL, &dateStarted, &item.DateFinished, &item.DateCreated)
	if err != nil {
		return Deploy{}, fmt.Errorf("create deploy: %w", err)
	}
	item.Name = nullableStringValue(name)
	item.URL = nullableStringValue(deployURL)
	item.DateStarted = nullableStringValue(dateStarted)
	return item, nil
}

func (r *Repository) ListDeploys(ctx context.Context, organizationRef string, version string) ([]Deploy, error) {
	releaseID, err := r.releaseID(ctx, organizationRef, version)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
SELECT id::text, environment, COALESCE(name, ''), COALESCE(url, ''), COALESCE(date_started::text, ''), date_finished, created_at
FROM release_deploys
WHERE release_id = $1::uuid
ORDER BY date_finished DESC`, releaseID)
	if err != nil {
		return nil, fmt.Errorf("list deploys: %w", err)
	}
	defer rows.Close()

	var items []Deploy
	for rows.Next() {
		var item Deploy
		var name string
		var deployURL string
		var dateStarted string
		if err := rows.Scan(&item.ID, &item.Environment, &name, &deployURL, &dateStarted, &item.DateFinished, &item.DateCreated); err != nil {
			return nil, fmt.Errorf("scan deploy: %w", err)
		}
		item.Name = nullableStringValue(name)
		item.URL = nullableStringValue(deployURL)
		item.DateStarted = nullableStringValue(dateStarted)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deploys: %w", err)
	}
	return items, nil
}

func (r *Repository) ensureRelease(ctx context.Context, organizationRef string, projectRef string, version string) (string, error) {
	release, err := r.CreateOrUpdate(ctx, organizationRef, CreateReleaseInput{
		Version:  version,
		Projects: optionalProject(projectRef),
	})
	if err != nil {
		return "", err
	}
	return release.ID, nil
}

func (r *Repository) releaseID(ctx context.Context, organizationRef string, version string) (string, error) {
	orgID, err := r.resolveOrganization(ctx, organizationRef)
	if err != nil {
		return "", err
	}
	var releaseID string
	err = r.db.QueryRow(ctx, `
SELECT id::text
FROM releases
WHERE organization_id = $1::uuid AND version = $2`, orgID, version).Scan(&releaseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrReleaseNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve release: %w", err)
	}
	return releaseID, nil
}

func (r *Repository) resolveOrganization(ctx context.Context, organizationRef string) (string, error) {
	var orgID string
	err := r.db.QueryRow(ctx, `
SELECT id::text
FROM organizations
WHERE id::text = $1 OR slug = $1
LIMIT 1`, organizationRef).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrOrganizationNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve organization: %w", err)
	}
	return orgID, nil
}

func (r *Repository) resolveProject(ctx context.Context, organizationID string, projectRef string) (string, error) {
	var projectID string
	err := r.db.QueryRow(ctx, `
SELECT id::text
FROM projects
WHERE organization_id = $1::uuid
  AND (id::text = $2 OR slug = $2)
LIMIT 1`, organizationID, projectRef).Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrProjectNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve project: %w", err)
	}
	return projectID, nil
}

func (r *Repository) optionalProjectID(ctx context.Context, organizationRef string, projectRef string) (string, error) {
	if projectRef == "" {
		return "", nil
	}
	orgID, err := r.resolveOrganization(ctx, organizationRef)
	if err != nil {
		return "", err
	}
	return r.resolveProject(ctx, orgID, projectRef)
}

func (r *Repository) withReleaseResponseDefaults(ctx context.Context, item Release, ref string, releaseURL string, dateReleased string) (Release, error) {
	projects, err := r.releaseProjects(ctx, item.ID)
	if err != nil {
		return Release{}, err
	}
	deploys, err := r.ListDeploysByReleaseID(ctx, item.ID)
	if err != nil {
		return Release{}, err
	}
	item.Authors = []any{}
	item.CommitCount = 0
	item.Data = map[string]any{}
	item.DateReleased = nullableStringValue(dateReleased)
	item.DeployCount = len(deploys)
	item.FirstEvent = nil
	item.LastCommit = nil
	item.LastDeploy = nil
	if len(deploys) > 0 {
		item.LastDeploy = deploys[0]
	}
	item.LastEvent = nil
	item.NewGroups = 0
	item.Owner = nil
	item.Projects = projects
	item.Ref = nullableStringValue(ref)
	item.ShortVersion = item.Version
	item.URL = nullableStringValue(releaseURL)
	return item, nil
}

func (r *Repository) ListDeploysByReleaseID(ctx context.Context, releaseID string) ([]Deploy, error) {
	rows, err := r.db.Query(ctx, `
SELECT id::text, environment, COALESCE(name, ''), COALESCE(url, ''), COALESCE(date_started::text, ''), date_finished, created_at
FROM release_deploys
WHERE release_id = $1::uuid
ORDER BY date_finished DESC`, releaseID)
	if err != nil {
		return nil, fmt.Errorf("list deploys by release: %w", err)
	}
	defer rows.Close()

	var items []Deploy
	for rows.Next() {
		var item Deploy
		var name string
		var deployURL string
		var dateStarted string
		if err := rows.Scan(&item.ID, &item.Environment, &name, &deployURL, &dateStarted, &item.DateFinished, &item.DateCreated); err != nil {
			return nil, fmt.Errorf("scan deploy: %w", err)
		}
		item.Name = nullableStringValue(name)
		item.URL = nullableStringValue(deployURL)
		item.DateStarted = nullableStringValue(dateStarted)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deploys: %w", err)
	}
	return items, nil
}

func (r *Repository) releaseProjects(ctx context.Context, releaseID string) ([]ReleaseProject, error) {
	rows, err := r.db.Query(ctx, `
SELECT p.name, p.slug
FROM release_projects rp
JOIN projects p ON p.id = rp.project_id
WHERE rp.release_id = $1::uuid
ORDER BY p.slug`, releaseID)
	if err != nil {
		return nil, fmt.Errorf("list release projects: %w", err)
	}
	defer rows.Close()

	var projects []ReleaseProject
	for rows.Next() {
		var project ReleaseProject
		if err := rows.Scan(&project.Name, &project.Slug); err != nil {
			return nil, fmt.Errorf("scan release project: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate release projects: %w", err)
	}
	return projects, nil
}

func nullableStringValue(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalProject(projectRef string) []string {
	if projectRef == "" {
		return nil
	}
	return []string{projectRef}
}
