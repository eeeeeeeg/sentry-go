CREATE DATABASE IF NOT EXISTS sentry;

CREATE TABLE IF NOT EXISTS sentry.events
(
    event_id UUID,
    project_id UUID,
    issue_id Nullable(UUID),
    timestamp DateTime64(3, 'UTC'),
    received_at DateTime64(3, 'UTC') DEFAULT now64(3),
    platform LowCardinality(String),
    runtime_name LowCardinality(String),
    runtime_version String,
    sdk_name LowCardinality(String),
    sdk_version String,
    level LowCardinality(String),
    message String,
    exception_type String,
    exception_value String,
    release String,
    environment LowCardinality(String),
    user_id String,
    tags String,
    contexts String,
    raw_event String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, timestamp, event_id)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS sentry.event_exceptions
(
    event_id UUID,
    project_id UUID,
    timestamp DateTime64(3, 'UTC'),
    type String,
    value String,
    stacktrace String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, timestamp, event_id);

CREATE TABLE IF NOT EXISTS sentry.event_breadcrumbs
(
    event_id UUID,
    project_id UUID,
    timestamp DateTime64(3, 'UTC'),
    category LowCardinality(String),
    level LowCardinality(String),
    message String,
    data String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, timestamp, event_id);

CREATE TABLE IF NOT EXISTS sentry.event_tags
(
    event_id UUID,
    project_id UUID,
    timestamp DateTime64(3, 'UTC'),
    key LowCardinality(String),
    value String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, key, value, timestamp, event_id);

CREATE TABLE IF NOT EXISTS sentry.event_users
(
    event_id UUID,
    project_id UUID,
    timestamp DateTime64(3, 'UTC'),
    user_id String,
    email String,
    username String,
    ip_address String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, user_id, timestamp, event_id);
