package replay

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrReplaySegmentNotFound = errors.New("replay recording segment not found")

type Repository struct {
	db *pgxpool.Pool
}

type RecordingSegment struct {
	ID          string    `json:"id"`
	ReplayID    string    `json:"replayId"`
	SegmentID   int       `json:"segmentId"`
	ProjectID   string    `json:"projectId"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	DateCreated time.Time `json:"dateCreated"`
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListRecordingSegments(ctx context.Context, organizationRef string, projectRef string, replayID string) ([]RecordingSegment, error) {
	replayID = canonicalID(replayID)
	rows, err := r.db.Query(ctx, `
SELECT ri.id::text,
       ri.replay_id,
       ri.segment_id,
       ri.project_id::text,
       ri.content_type,
       ri.size_bytes,
       ri.received_at
FROM replay_items ri
JOIN projects p ON p.id = ri.project_id
JOIN organizations o ON o.id = p.organization_id
WHERE (o.id::text = $1 OR o.slug = $1)
  AND (p.id::text = $2 OR p.slug = $2)
  AND ri.replay_id = $3
  AND ri.item_type = 'replay_recording'
ORDER BY ri.segment_id ASC, ri.received_at ASC`, organizationRef, projectRef, replayID)
	if err != nil {
		return nil, fmt.Errorf("list replay recording segments: %w", err)
	}
	defer rows.Close()

	var items []RecordingSegment
	for rows.Next() {
		var item RecordingSegment
		if err := rows.Scan(&item.ID, &item.ReplayID, &item.SegmentID, &item.ProjectID, &item.ContentType, &item.Size, &item.DateCreated); err != nil {
			return nil, fmt.Errorf("scan replay recording segment: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate replay recording segments: %w", err)
	}
	return items, nil
}

func (r *Repository) GetRecordingSegment(ctx context.Context, organizationRef string, projectRef string, replayID string, segmentID int) (RecordingSegment, []byte, error) {
	replayID = canonicalID(replayID)
	var item RecordingSegment
	var content []byte
	err := r.db.QueryRow(ctx, `
SELECT ri.id::text,
       ri.replay_id,
       ri.segment_id,
       ri.project_id::text,
       ri.content_type,
       ri.size_bytes,
       ri.received_at,
       ri.payload
FROM replay_items ri
JOIN projects p ON p.id = ri.project_id
JOIN organizations o ON o.id = p.organization_id
WHERE (o.id::text = $1 OR o.slug = $1)
  AND (p.id::text = $2 OR p.slug = $2)
  AND ri.replay_id = $3
  AND ri.segment_id = $4
  AND ri.item_type = 'replay_recording'
ORDER BY ri.received_at ASC
LIMIT 1`, organizationRef, projectRef, replayID, segmentID).Scan(
		&item.ID,
		&item.ReplayID,
		&item.SegmentID,
		&item.ProjectID,
		&item.ContentType,
		&item.Size,
		&item.DateCreated,
		&content,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RecordingSegment{}, nil, ErrReplaySegmentNotFound
	}
	if err != nil {
		return RecordingSegment{}, nil, fmt.Errorf("get replay recording segment: %w", err)
	}
	return item, content, nil
}
