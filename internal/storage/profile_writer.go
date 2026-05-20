package storage

import (
	"context"
	"database/sql"
	"fmt"

	"sentry-lite/internal/profile"
)

type ProfileWriter struct {
	db *sql.DB
}

func NewProfileWriter(db *sql.DB) *ProfileWriter {
	return &ProfileWriter{db: db}
}

func (w *ProfileWriter) InsertProfile(ctx context.Context, item profile.ProfileRecord) error {
	_, err := w.db.ExecContext(ctx, `
INSERT INTO sentry.profiles
(
    profile_id,
    event_id,
    organization_id,
    project_id,
    project_key_id,
    trace_id,
    transaction_id,
    transaction,
    platform,
    version,
    release,
    environment,
    received_at,
    sdk_name,
    sdk_version,
    item_type,
    duration_ns,
    sample_count,
    thread_count,
    raw_profile
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ProfileID,
		item.EventID,
		item.OrganizationID,
		item.ProjectID,
		item.ProjectKeyID,
		item.TraceID,
		item.TransactionID,
		item.Transaction,
		item.Platform,
		item.Version,
		item.Release,
		item.Environment,
		item.ReceivedAt,
		item.SDKName,
		item.SDKVersion,
		item.ItemType,
		item.DurationNS,
		item.SampleCount,
		item.ThreadCount,
		string(item.RawProfile),
	)
	if err != nil {
		return fmt.Errorf("insert profile: %w", err)
	}
	return nil
}
