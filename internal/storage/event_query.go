package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type NamedVersion struct {
	Name    string         `json:"name,omitempty"`
	Version string         `json:"version,omitempty"`
	Family  string         `json:"family,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

type EventBreadcrumb struct {
	Timestamp string         `json:"timestamp,omitempty"`
	Type      string         `json:"type,omitempty"`
	Category  string         `json:"category,omitempty"`
	Level     string         `json:"level,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

type Event struct {
	EventID        string            `json:"event_id"`
	ProjectID      string            `json:"project_id"`
	IssueID        string            `json:"issue_id,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
	ReceivedAt     time.Time         `json:"received_at"`
	Platform       string            `json:"platform"`
	Level          string            `json:"level"`
	Message        string            `json:"message"`
	ExceptionType  string            `json:"exception_type"`
	ExceptionValue string            `json:"exception_value"`
	Release        string            `json:"release,omitempty"`
	Environment    string            `json:"environment,omitempty"`
	UserID         string            `json:"user_id,omitempty"`
	Tags           string            `json:"tags"`
	Contexts       string            `json:"contexts"`
	RawEvent       string            `json:"raw_event,omitempty"`
	RuntimeName    string            `json:"runtime_name,omitempty"`
	RuntimeVersion string            `json:"runtime_version,omitempty"`
	SDKName        string            `json:"sdk_name,omitempty"`
	SDKVersion     string            `json:"sdk_version,omitempty"`
	Browser        *NamedVersion     `json:"browser,omitempty"`
	OS             *NamedVersion     `json:"os,omitempty"`
	Device         *NamedVersion     `json:"device,omitempty"`
	Culture        map[string]any    `json:"culture,omitempty"`
	Trace          map[string]any    `json:"trace,omitempty"`
	Request        map[string]any    `json:"request,omitempty"`
	User           map[string]any    `json:"user,omitempty"`
	Breadcrumbs    []EventBreadcrumb `json:"breadcrumbs,omitempty"`
}

type EventQuery struct {
	ProjectID   string
	IssueID     string
	Level       string
	Environment string
	Release     string
	Since       time.Time
	Until       time.Time
	Limit       int
	Offset      int
}

type EventQuerier struct {
	db *sql.DB
}

func NewEventQuerier(db *sql.DB) *EventQuerier {
	return &EventQuerier{db: db}
}

func (q *EventQuerier) List(ctx context.Context, query EventQuery) ([]Event, int, error) {
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	whereSQL, args := eventWhere(query)
	countArgs := append([]any{}, args...)
	var totalValue uint64
	if err := q.db.QueryRowContext(ctx, "SELECT count() FROM sentry.events "+whereSQL, countArgs...).Scan(&totalValue); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	listArgs := append(args, limit, query.Offset)
	rows, err := q.db.QueryContext(ctx, `
SELECT
    toString(event_id),
    toString(project_id),
    ifNull(toString(issue_id), ''),
    timestamp,
    received_at,
    platform,
    level,
    message,
    exception_type,
    exception_value,
    release,
    environment,
    user_id,
    tags,
    contexts,
    runtime_name,
    runtime_version,
    sdk_name,
    sdk_version
FROM sentry.events
`+whereSQL+`
ORDER BY timestamp DESC
LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.EventID,
			&event.ProjectID,
			&event.IssueID,
			&event.Timestamp,
			&event.ReceivedAt,
			&event.Platform,
			&event.Level,
			&event.Message,
			&event.ExceptionType,
			&event.ExceptionValue,
			&event.Release,
			&event.Environment,
			&event.UserID,
			&event.Tags,
			&event.Contexts,
			&event.RuntimeName,
			&event.RuntimeVersion,
			&event.SDKName,
			&event.SDKVersion,
		); err != nil {
			return nil, 0, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate events: %w", err)
	}
	return events, int(totalValue), nil
}

func (q *EventQuerier) Get(ctx context.Context, eventID string) (Event, error) {
	var event Event
	err := q.db.QueryRowContext(ctx, `
SELECT
    toString(event_id),
    toString(project_id),
    ifNull(toString(issue_id), ''),
    timestamp,
    received_at,
    platform,
    level,
    message,
    exception_type,
    exception_value,
    release,
    environment,
    user_id,
    tags,
    contexts,
    raw_event,
    runtime_name,
    runtime_version,
    sdk_name,
    sdk_version
FROM sentry.events
WHERE event_id = toUUID(?)
LIMIT 1`, eventID).Scan(
		&event.EventID,
		&event.ProjectID,
		&event.IssueID,
		&event.Timestamp,
		&event.ReceivedAt,
		&event.Platform,
		&event.Level,
		&event.Message,
		&event.ExceptionType,
		&event.ExceptionValue,
		&event.Release,
		&event.Environment,
		&event.UserID,
		&event.Tags,
		&event.Contexts,
		&event.RawEvent,
		&event.RuntimeName,
		&event.RuntimeVersion,
		&event.SDKName,
		&event.SDKVersion,
	)
	if err != nil {
		return Event{}, fmt.Errorf("get event: %w", err)
	}
	event.enrich()
	return event, nil
}

func (e *Event) enrich() {
	payload := map[string]any{}
	if e.RawEvent != "" {
		_ = json.Unmarshal([]byte(e.RawEvent), &payload)
	}

	contexts := map[string]any{}
	if e.Contexts != "" {
		_ = json.Unmarshal([]byte(e.Contexts), &contexts)
	}
	if rawContexts, ok := payload["contexts"].(map[string]any); ok {
		for key, value := range rawContexts {
			contexts[key] = value
		}
	}

	e.Browser = namedContext(contexts, "browser")
	e.OS = namedContext(contexts, "os")
	e.Device = namedContext(contexts, "device")
	e.Culture = mapContext(contexts, "culture")
	e.Trace = mapContext(contexts, "trace")
	e.Request = mapPayload(payload, "request")
	e.User = mapPayload(payload, "user")
	e.Breadcrumbs = parseBreadcrumbs(payload["breadcrumbs"])
	e.fillClientFromTags()
	e.fillClientFromUserAgent()

	if sdk, ok := payload["sdk"].(map[string]any); ok && e.SDKName == "" && e.SDKVersion == "" {
		e.SDKName = stringValue(sdk["name"])
		e.SDKVersion = stringValue(sdk["version"])
	}
	if runtime, ok := payload["runtime"].(map[string]any); ok && e.RuntimeName == "" && e.RuntimeVersion == "" {
		e.RuntimeName = stringValue(runtime["name"])
		e.RuntimeVersion = stringValue(runtime["version"])
	}
}

func (e *Event) fillClientFromTags() {
	tags := map[string]any{}
	if e.Tags == "" {
		return
	}
	if err := json.Unmarshal([]byte(e.Tags), &tags); err != nil {
		return
	}
	if e.Browser == nil {
		if name := stringValue(tags["browser.name"]); name != "" {
			e.Browser = &NamedVersion{Name: name, Version: browserVersionFromTag(stringValue(tags["browser"]), name)}
		}
	}
	if e.OS == nil {
		if name := stringValue(tags["os.name"]); name != "" {
			e.OS = &NamedVersion{Name: name}
		}
	}
}

func (e *Event) fillClientFromUserAgent() {
	userAgent := requestUserAgent(e.Request)
	if userAgent == "" {
		return
	}
	if e.Browser == nil {
		e.Browser = browserFromUserAgent(userAgent)
	}
	if e.OS == nil {
		e.OS = osFromUserAgent(userAgent)
	}
	if e.Device == nil {
		e.Device = deviceFromUserAgent(userAgent)
	}
}

func requestUserAgent(request map[string]any) string {
	headers, ok := request["headers"].(map[string]any)
	if !ok {
		return ""
	}
	for key, value := range headers {
		if strings.EqualFold(key, "user-agent") {
			return stringValue(value)
		}
	}
	return ""
}

func browserFromUserAgent(userAgent string) *NamedVersion {
	switch {
	case strings.Contains(userAgent, "Edg/"):
		return &NamedVersion{Name: "Edge", Version: tokenVersion(userAgent, "Edg/")}
	case strings.Contains(userAgent, "Chrome/"):
		return &NamedVersion{Name: "Chrome", Version: tokenVersion(userAgent, "Chrome/")}
	case strings.Contains(userAgent, "Firefox/"):
		return &NamedVersion{Name: "Firefox", Version: tokenVersion(userAgent, "Firefox/")}
	case strings.Contains(userAgent, "Safari/") && strings.Contains(userAgent, "Version/"):
		return &NamedVersion{Name: "Safari", Version: tokenVersion(userAgent, "Version/")}
	default:
		return nil
	}
}

func osFromUserAgent(userAgent string) *NamedVersion {
	switch {
	case strings.Contains(userAgent, "Windows NT 10.0"):
		return &NamedVersion{Name: "Windows", Version: "10"}
	case strings.Contains(userAgent, "Windows NT 6.3"):
		return &NamedVersion{Name: "Windows", Version: "8.1"}
	case strings.Contains(userAgent, "Windows NT 6.2"):
		return &NamedVersion{Name: "Windows", Version: "8"}
	case strings.Contains(userAgent, "Windows NT 6.1"):
		return &NamedVersion{Name: "Windows", Version: "7"}
	case strings.Contains(userAgent, "Mac OS X"):
		return &NamedVersion{Name: "macOS", Version: strings.ReplaceAll(tokenVersion(userAgent, "Mac OS X "), "_", ".")}
	case strings.Contains(userAgent, "Android"):
		return &NamedVersion{Name: "Android", Version: tokenVersion(userAgent, "Android ")}
	case strings.Contains(userAgent, "iPhone OS"):
		return &NamedVersion{Name: "iOS", Version: strings.ReplaceAll(tokenVersion(userAgent, "iPhone OS "), "_", ".")}
	case strings.Contains(userAgent, "Linux"):
		return &NamedVersion{Name: "Linux"}
	default:
		return nil
	}
}

func deviceFromUserAgent(userAgent string) *NamedVersion {
	switch {
	case strings.Contains(userAgent, "Mobile") || strings.Contains(userAgent, "iPhone"):
		return &NamedVersion{Family: "Mobile"}
	case strings.Contains(userAgent, "iPad") || strings.Contains(userAgent, "Tablet"):
		return &NamedVersion{Family: "Tablet"}
	default:
		return &NamedVersion{Family: "Other"}
	}
}

func browserVersionFromTag(tag string, name string) string {
	prefix := name + " "
	if strings.HasPrefix(tag, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(tag, prefix))
	}
	return ""
}

func tokenVersion(value string, token string) string {
	index := strings.Index(value, token)
	if index < 0 {
		return ""
	}
	rest := value[index+len(token):]
	for i, r := range rest {
		if r == ' ' || r == ';' || r == ')' {
			return rest[:i]
		}
	}
	return rest
}

func namedContext(contexts map[string]any, key string) *NamedVersion {
	data := mapContext(contexts, key)
	if len(data) == 0 {
		return nil
	}
	item := &NamedVersion{
		Name:    stringValue(data["name"]),
		Version: stringValue(data["version"]),
		Family:  stringValue(data["family"]),
		Data:    data,
	}
	return item
}

func mapContext(contexts map[string]any, key string) map[string]any {
	value, ok := contexts[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func mapPayload(payload map[string]any, key string) map[string]any {
	value, ok := payload[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func parseBreadcrumbs(value any) []EventBreadcrumb {
	var values []any
	switch typed := value.(type) {
	case []any:
		values = typed
	case map[string]any:
		values, _ = typed["values"].([]any)
	}
	if len(values) == 0 {
		return nil
	}

	breadcrumbs := make([]EventBreadcrumb, 0, len(values))
	for _, raw := range values {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		data, _ := item["data"].(map[string]any)
		breadcrumbs = append(breadcrumbs, EventBreadcrumb{
			Timestamp: timestampValue(item["timestamp"]),
			Type:      stringValue(item["type"]),
			Category:  stringValue(item["category"]),
			Level:     stringValue(item["level"]),
			Message:   stringValue(item["message"]),
			Data:      data,
		})
	}
	return breadcrumbs
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%v", typed)
	default:
		return ""
	}
}

func timestampValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		seconds := int64(typed)
		nanos := int64((typed - float64(seconds)) * 1e9)
		return time.Unix(seconds, nanos).UTC().Format(time.RFC3339Nano)
	default:
		return ""
	}
}

func eventWhere(query EventQuery) (string, []any) {
	conditions := []string{"toString(project_id) = ?"}
	args := []any{query.ProjectID}

	if query.IssueID != "" {
		conditions = append(conditions, "ifNull(toString(issue_id), '') = ?")
		args = append(args, query.IssueID)
	}
	if query.Level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, query.Level)
	}
	if query.Environment != "" {
		conditions = append(conditions, "environment = ?")
		args = append(args, query.Environment)
	}
	if query.Release != "" {
		conditions = append(conditions, "release = ?")
		args = append(args, query.Release)
	}
	if !query.Since.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, query.Since)
	}
	if !query.Until.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, query.Until)
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}
