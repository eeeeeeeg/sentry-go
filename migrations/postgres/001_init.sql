CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE SEQUENCE IF NOT EXISTS sentry_project_id_seq START 2;

CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, slug)
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sentry_project_id TEXT NOT NULL DEFAULT nextval('sentry_project_id_seq')::text,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    platform TEXT NOT NULL DEFAULT 'javascript',
    status TEXT NOT NULL DEFAULT 'active',
    sample_rate NUMERIC(5, 4) NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (sentry_project_id),
    UNIQUE (organization_id, slug),
    CHECK (status IN ('active', 'disabled')),
    CHECK (sample_rate >= 0 AND sample_rate <= 1)
);

CREATE TABLE IF NOT EXISTS project_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    rate_limit_per_minute INTEGER NOT NULL DEFAULT 6000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status IN ('active', 'disabled')),
    CHECK (rate_limit_per_minute > 0)
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    title TEXT NOT NULL,
    culprit TEXT,
    level TEXT NOT NULL DEFAULT 'error',
    status TEXT NOT NULL DEFAULT 'unresolved',
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    event_count BIGINT NOT NULL DEFAULT 0,
    user_count BIGINT NOT NULL DEFAULT 0,
    release TEXT,
    environment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, fingerprint),
    CHECK (status IN ('unresolved', 'resolved', 'ignored'))
);

CREATE TABLE IF NOT EXISTS issue_status_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    old_status TEXT NOT NULL,
    new_status TEXT NOT NULL,
    reason TEXT,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (old_status IN ('unresolved', 'resolved', 'ignored')),
    CHECK (new_status IN ('unresolved', 'resolved', 'ignored'))
);

CREATE TABLE IF NOT EXISTS releases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    ref TEXT,
    url TEXT,
    date_released TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, version)
);

CREATE TABLE IF NOT EXISTS release_projects (
    release_id UUID NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (release_id, project_id)
);

CREATE TABLE IF NOT EXISTS release_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id UUID NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    dist TEXT,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    sha1 TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    content BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS release_deploys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id UUID NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    environment TEXT NOT NULL,
    name TEXT,
    url TEXT,
    date_started TIMESTAMPTZ,
    date_finished TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    event_type TEXT NOT NULL DEFAULT 'new_issue',
    channel TEXT NOT NULL DEFAULT 'webhook',
    webhook_url TEXT,
    min_level TEXT NOT NULL DEFAULT 'error',
    threshold_count INTEGER NOT NULL DEFAULT 1,
    window_seconds INTEGER NOT NULL DEFAULT 300,
    status TEXT NOT NULL DEFAULT 'active',
    cooldown_seconds INTEGER NOT NULL DEFAULT 300,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (event_type IN ('new_issue', 'regression', 'frequency')),
    CHECK (channel IN ('webhook')),
    CHECK (status IN ('active', 'disabled')),
    CHECK (threshold_count > 0),
    CHECK (window_seconds > 0),
    CHECK (cooldown_seconds >= 0)
);

CREATE TABLE IF NOT EXISTS alert_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    channel TEXT NOT NULL,
    status TEXT NOT NULL,
    error TEXT,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status IN ('sent', 'failed', 'suppressed'))
);

CREATE TABLE IF NOT EXISTS event_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    project_key_id UUID REFERENCES project_keys(id) ON DELETE SET NULL,
    event_id TEXT,
    message_id TEXT NOT NULL UNIQUE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    attachment_type TEXT,
    sha1 TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    content BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS replay_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    project_key_id UUID REFERENCES project_keys(id) ON DELETE SET NULL,
    replay_id TEXT NOT NULL,
    event_id TEXT,
    trace_id TEXT,
    transaction_id TEXT,
    segment_id INTEGER NOT NULL DEFAULT 0,
    item_type TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sdk_name TEXT,
    sdk_version TEXT,
    content_type TEXT NOT NULL DEFAULT 'application/json',
    size_bytes BIGINT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    payload BYTEA NOT NULL,
    message_id TEXT NOT NULL UNIQUE,
    CHECK (item_type IN ('replay_event', 'replay_recording'))
);

CREATE INDEX IF NOT EXISTS idx_project_keys_project_id ON project_keys(project_id);
CREATE INDEX IF NOT EXISTS idx_issues_project_status_last_seen ON issues(project_id, status, last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_issues_project_environment ON issues(project_id, environment);
CREATE INDEX IF NOT EXISTS idx_issues_project_release ON issues(project_id, release);
CREATE INDEX IF NOT EXISTS idx_issue_status_changes_issue_created ON issue_status_changes(issue_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_releases_org_created ON releases(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_release_files_release_created ON release_files(release_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_release_files_unique ON release_files(release_id, (COALESCE(project_id, '00000000-0000-0000-0000-000000000000'::uuid)), name, (COALESCE(dist, '')));
CREATE INDEX IF NOT EXISTS idx_release_deploys_release_finished ON release_deploys(release_id, date_finished DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_project_event ON alerts(project_id, event_type, status);
CREATE INDEX IF NOT EXISTS idx_alert_deliveries_issue_created ON alert_deliveries(issue_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_event_attachments_project_event ON event_attachments(project_id, event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_replay_items_project_replay ON replay_items(project_id, replay_id, segment_id, item_type);
CREATE INDEX IF NOT EXISTS idx_replay_items_project_received ON replay_items(project_id, received_at DESC);

INSERT INTO organizations (slug, name)
VALUES ('demo', 'Demo Organization')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO projects (organization_id, sentry_project_id, slug, name, platform)
SELECT id, '1', 'web', 'Demo Web Project', 'javascript'
FROM organizations
WHERE slug = 'demo'
ON CONFLICT (organization_id, slug) DO NOTHING;

INSERT INTO project_keys (project_id, public_key, name)
SELECT id, 'demo-public-key', 'Default DSN Key'
FROM projects
WHERE slug = 'web'
ON CONFLICT (public_key) DO NOTHING;

INSERT INTO api_tokens (organization_id, name, token_hash, scopes)
SELECT id,
       'Demo API Token',
       '9477b34a9c255f76f79d282640e9f9d02f1b32a370408fdac63538ce33a788ed',
       ARRAY['project:read', 'project:write', 'project:admin', 'project:releases', 'org:ci', 'event:read', 'event:write', 'event:admin']
FROM organizations
WHERE slug = 'demo'
ON CONFLICT (token_hash) DO UPDATE
SET scopes = (
    SELECT ARRAY(
        SELECT DISTINCT scope
        FROM unnest(api_tokens.scopes || EXCLUDED.scopes) AS scope
        ORDER BY scope
    )
);
