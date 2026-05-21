package platform

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ensurePostgresCompatibility(ctx context.Context, db *pgxpool.Pool) error {
	statements := []string{
		`CREATE SEQUENCE IF NOT EXISTS sentry_project_id_seq START 2`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS sentry_project_id TEXT`,
		`
WITH numbered AS (
    SELECT p.id,
           CASE
               WHEN o.slug = 'demo' AND p.slug = 'web' THEN '1'
               ELSE (row_number() OVER (ORDER BY p.created_at, p.id) + 1)::text
           END AS generated_id
    FROM projects p
    JOIN organizations o ON o.id = p.organization_id
    WHERE p.sentry_project_id IS NULL OR p.sentry_project_id = ''
)
UPDATE projects p
SET sentry_project_id = numbered.generated_id
FROM numbered
WHERE p.id = numbered.id`,
		`ALTER TABLE projects ALTER COLUMN sentry_project_id SET DEFAULT nextval('sentry_project_id_seq')::text`,
		`
SELECT setval(
    'sentry_project_id_seq',
    GREATEST(
        (SELECT COALESCE(MAX(sentry_project_id::bigint), 1) FROM projects WHERE sentry_project_id ~ '^[0-9]+$'),
        1
    ) + 1,
    false
)`,
		`ALTER TABLE projects ALTER COLUMN sentry_project_id SET NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_sentry_project_id ON projects(sentry_project_id)`,
		`
UPDATE project_keys
SET public_key = '0123456789abcdef0123456789abcdef'
WHERE public_key = 'demo-public-key'
  AND NOT EXISTS (
      SELECT 1
      FROM project_keys existing
      WHERE existing.public_key = '0123456789abcdef0123456789abcdef'
  )`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("postgres compatibility migration failed: %w", err)
		}
	}
	return nil
}
