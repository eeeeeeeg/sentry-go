package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv   string
	HTTPAddr string
	LogLevel string

	PostgresDSN string

	ClickHouseAddr     string
	ClickHouseDatabase string
	ClickHouseUsername string
	ClickHousePassword string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	NATSURL string

	RawEventStream         string
	RawEventSubject        string
	RawTransactionSubject  string
	RawSessionSubject      string
	RawAttachmentSubject   string
	RawProfileSubject      string
	RawReplaySubject       string
	RawOutcomeSubject      string
	UnsupportedItemSubject string
	NormalizedEventSubject string
	GroupedEventSubject    string
	AlertEventSubject      string
	DeadLetterSubject      string
	NormalizeConsumer      string
	TransactionConsumer    string
	GroupingConsumer       string
	EventWriterConsumer    string
	AttachmentConsumer     string
	ProfileConsumer        string
	ReplayConsumer         string
	SessionConsumer        string
	OutcomeConsumer        string
	AlertConsumer          string
	AlertSuppressWindow    time.Duration

	MaxEnvelopeBytes    int64
	MaxReleaseFileBytes int64
	DefaultRateLimit    int64
	RateLimitWindow     time.Duration

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
}

func Load() (Config, error) {
	redisDB, err := strconv.Atoi(getenv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, err
	}
	maxEnvelopeBytes, err := strconv.ParseInt(getenv("MAX_ENVELOPE_BYTES", "262144"), 10, 64)
	if err != nil {
		return Config{}, err
	}
	maxReleaseFileBytes, err := strconv.ParseInt(getenv("MAX_RELEASE_FILE_BYTES", "20971520"), 10, 64)
	if err != nil {
		return Config{}, err
	}
	defaultRateLimit, err := strconv.ParseInt(getenv("DEFAULT_RATE_LIMIT_PER_MINUTE", "6000"), 10, 64)
	if err != nil {
		return Config{}, err
	}
	alertSuppressSeconds, err := strconv.ParseInt(getenv("ALERT_SUPPRESS_SECONDS", "300"), 10, 64)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:   getenv("APP_ENV", "local"),
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
		LogLevel: getenv("LOG_LEVEL", "info"),

		PostgresDSN: getenv("POSTGRES_DSN", "postgres://sentry:sentry@localhost:5432/sentry?sslmode=disable"),

		ClickHouseAddr:     getenv("CLICKHOUSE_ADDR", "localhost:9000"),
		ClickHouseDatabase: getenv("CLICKHOUSE_DATABASE", "sentry"),
		ClickHouseUsername: getenv("CLICKHOUSE_USERNAME", "default"),
		ClickHousePassword: getenv("CLICKHOUSE_PASSWORD", ""),

		RedisAddr:     getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getenv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,

		NATSURL: getenv("NATS_URL", "nats://localhost:4222"),

		RawEventStream:         getenv("RAW_EVENT_STREAM", "EVENTS"),
		RawEventSubject:        getenv("RAW_EVENT_SUBJECT", "events.raw"),
		RawTransactionSubject:  getenv("RAW_TRANSACTION_SUBJECT", "transactions.raw"),
		RawSessionSubject:      getenv("RAW_SESSION_SUBJECT", "sessions.raw"),
		RawAttachmentSubject:   getenv("RAW_ATTACHMENT_SUBJECT", "attachments.raw"),
		RawProfileSubject:      getenv("RAW_PROFILE_SUBJECT", "profiles.raw"),
		RawReplaySubject:       getenv("RAW_REPLAY_SUBJECT", "replays.raw"),
		RawOutcomeSubject:      getenv("RAW_OUTCOME_SUBJECT", "outcomes.raw"),
		UnsupportedItemSubject: getenv("UNSUPPORTED_ITEM_SUBJECT", "envelopes.unsupported"),
		NormalizedEventSubject: getenv("NORMALIZED_EVENT_SUBJECT", "events.normalized"),
		GroupedEventSubject:    getenv("GROUPED_EVENT_SUBJECT", "events.grouped"),
		AlertEventSubject:      getenv("ALERT_EVENT_SUBJECT", "alerts.triggered"),
		DeadLetterSubject:      getenv("DEAD_LETTER_SUBJECT", "events.dlq"),
		NormalizeConsumer:      getenv("NORMALIZE_CONSUMER", "worker-normalize"),
		TransactionConsumer:    getenv("TRANSACTION_CONSUMER", "worker-transaction"),
		GroupingConsumer:       getenv("GROUPING_CONSUMER", "worker-grouping"),
		EventWriterConsumer:    getenv("EVENT_WRITER_CONSUMER", "worker-event-writer"),
		AttachmentConsumer:     getenv("ATTACHMENT_CONSUMER", "worker-attachment"),
		ProfileConsumer:        getenv("PROFILE_CONSUMER", "worker-profile"),
		ReplayConsumer:         getenv("REPLAY_CONSUMER", "worker-replay"),
		SessionConsumer:        getenv("SESSION_CONSUMER", "worker-session"),
		OutcomeConsumer:        getenv("OUTCOME_CONSUMER", "worker-outcome"),
		AlertConsumer:          getenv("ALERT_CONSUMER", "worker-alert"),

		MaxEnvelopeBytes:    maxEnvelopeBytes,
		MaxReleaseFileBytes: maxReleaseFileBytes,
		DefaultRateLimit:    defaultRateLimit,
		RateLimitWindow:     time.Minute,
		AlertSuppressWindow: time.Duration(alertSuppressSeconds) * time.Second,

		MinIOEndpoint:  getenv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getenv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getenv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getenv("MINIO_BUCKET", "sentry-artifacts"),
	}, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
