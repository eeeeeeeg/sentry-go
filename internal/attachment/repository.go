package attachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAttachmentNotFound = errors.New("attachment not found")

type Repository struct {
	db *pgxpool.Pool
}

type Item struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	ContentType    string    `json:"contentType"`
	Size           int64     `json:"size"`
	SHA1           string    `json:"sha1"`
	EventID        string    `json:"eventID"`
	DateCreated    time.Time `json:"dateCreated"`
	AttachmentType string    `json:"attachmentType,omitempty"`
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListForEvent(ctx context.Context, organizationRef string, projectRef string, eventID string) ([]Item, error) {
	rows, err := r.db.Query(ctx, `
SELECT a.id::text,
       a.filename,
       COALESCE(a.attachment_type, 'event.attachment'),
       a.content_type,
       a.size_bytes,
       a.sha1,
       COALESCE(a.event_id, ''),
       a.created_at
FROM event_attachments a
JOIN projects p ON p.id = a.project_id
JOIN organizations o ON o.id = p.organization_id
WHERE (o.id::text = $1 OR o.slug = $1)
  AND (p.id::text = $2 OR p.slug = $2)
  AND a.event_id = $3
ORDER BY a.created_at DESC`, organizationRef, projectRef, eventID)
	if err != nil {
		return nil, fmt.Errorf("list event attachments: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.ContentType, &item.Size, &item.SHA1, &item.EventID, &item.DateCreated); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		item.AttachmentType = item.Type
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachments: %w", err)
	}
	return items, nil
}

func (r *Repository) GetForEvent(ctx context.Context, organizationRef string, projectRef string, eventID string, attachmentID string) (Item, []byte, error) {
	var item Item
	var content []byte
	err := r.db.QueryRow(ctx, `
SELECT a.id::text,
       a.filename,
       COALESCE(a.attachment_type, 'event.attachment'),
       a.content_type,
       a.size_bytes,
       a.sha1,
       COALESCE(a.event_id, ''),
       a.created_at,
       a.content
FROM event_attachments a
JOIN projects p ON p.id = a.project_id
JOIN organizations o ON o.id = p.organization_id
WHERE (o.id::text = $1 OR o.slug = $1)
  AND (p.id::text = $2 OR p.slug = $2)
  AND a.event_id = $3
  AND a.id::text = $4
LIMIT 1`, organizationRef, projectRef, eventID, attachmentID).Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&item.ContentType,
		&item.Size,
		&item.SHA1,
		&item.EventID,
		&item.DateCreated,
		&content,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, nil, ErrAttachmentNotFound
	}
	if err != nil {
		return Item{}, nil, fmt.Errorf("get event attachment: %w", err)
	}
	item.AttachmentType = item.Type
	return item, content, nil
}

func (r *Repository) DeleteForEvent(ctx context.Context, organizationRef string, projectRef string, eventID string, attachmentID string) error {
	tag, err := r.db.Exec(ctx, `
DELETE FROM event_attachments a
USING projects p, organizations o
WHERE p.id = a.project_id
  AND o.id = p.organization_id
  AND (o.id::text = $1 OR o.slug = $1)
  AND (p.id::text = $2 OR p.slug = $2)
  AND a.event_id = $3
  AND a.id::text = $4`, organizationRef, projectRef, eventID, attachmentID)
	if err != nil {
		return fmt.Errorf("delete event attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}
