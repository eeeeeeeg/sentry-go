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

CREATE TABLE IF NOT EXISTS sentry.outcomes
(
    project_id UUID,
    project_key_id UUID,
    event_id String,
    timestamp DateTime64(3, 'UTC'),
    received_at DateTime64(3, 'UTC') DEFAULT now64(3),
    category LowCardinality(String),
    reason LowCardinality(String),
    quantity UInt64,
    source LowCardinality(String),
    sdk_name LowCardinality(String),
    sdk_version String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, category, reason, timestamp);

CREATE TABLE IF NOT EXISTS sentry.sessions
(
    project_id UUID,
    project_key_id UUID,
    session_id String,
    distinct_id_hash String,
    started_at DateTime64(3, 'UTC'),
    bucket DateTime64(3, 'UTC'),
    timestamp DateTime64(3, 'UTC'),
    received_at DateTime64(3, 'UTC') DEFAULT now64(3),
    release String,
    environment LowCardinality(String),
    status LowCardinality(String),
    init UInt8,
    sequence Float64,
    errors UInt64,
    duration Float64,
    quantity UInt64,
    abnormal_mechanism LowCardinality(String),
    source LowCardinality(String),
    sdk_name LowCardinality(String),
    sdk_version String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(bucket)
ORDER BY (project_id, release, environment, bucket, status);

CREATE TABLE IF NOT EXISTS sentry.transactions
(
    event_id UUID,
    organization_id UUID,
    project_id UUID,
    project_key_id UUID,
    trace_id String,
    span_id String,
    parent_span_id String,
    transaction String,
    source LowCardinality(String),
    operation LowCardinality(String),
    status LowCardinality(String),
    start_timestamp DateTime64(3, 'UTC'),
    end_timestamp DateTime64(3, 'UTC'),
    duration_ms Float64,
    received_at DateTime64(3, 'UTC') DEFAULT now64(3),
    platform LowCardinality(String),
    release String,
    environment LowCardinality(String),
    sdk_name LowCardinality(String),
    sdk_version String,
    span_count UInt64,
    measurements String,
    contexts String,
    tags String,
    raw_transaction String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(start_timestamp)
ORDER BY (project_id, transaction, start_timestamp, event_id);

CREATE TABLE IF NOT EXISTS sentry.spans
(
    event_id UUID,
    project_id UUID,
    trace_id String,
    span_id String,
    parent_span_id String,
    operation LowCardinality(String),
    description String,
    status LowCardinality(String),
    start_timestamp DateTime64(3, 'UTC'),
    end_timestamp DateTime64(3, 'UTC'),
    duration_ms Float64,
    data String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(start_timestamp)
ORDER BY (project_id, trace_id, start_timestamp, span_id);

CREATE TABLE IF NOT EXISTS sentry.profiles
(
    profile_id String,
    event_id String,
    organization_id UUID,
    project_id UUID,
    project_key_id UUID,
    trace_id String,
    transaction_id String,
    transaction String,
    platform LowCardinality(String),
    version String,
    release String,
    environment LowCardinality(String),
    received_at DateTime64(3, 'UTC') DEFAULT now64(3),
    sdk_name LowCardinality(String),
    sdk_version String,
    item_type LowCardinality(String),
    duration_ns UInt64,
    sample_count UInt64,
    thread_count UInt64,
    raw_profile String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(received_at)
ORDER BY (project_id, transaction, received_at, profile_id);
