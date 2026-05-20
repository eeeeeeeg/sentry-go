package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"sentry-lite/internal/alert"
	"sentry-lite/internal/issue"
	"sentry-lite/internal/project"
	"sentry-lite/internal/storage"
)

type queryHandler struct {
	projects *project.Repository
	issues   *issue.Repository
	events   *storage.EventQuerier
	stats    *storage.StatsQuerier
	alerts   *alert.Repository
}

func (h queryHandler) register(r chi.Router) {
	r.Get("/api/projects/{project_id}/issues", h.listIssues)
	r.Get("/api/issues/{issue_id}", h.getIssue)
	r.Get("/api/issues/{issue_id}/status-changes", h.listIssueStatusChanges)
	r.Patch("/api/issues/{issue_id}/status", h.updateIssueStatus)
	r.Get("/api/projects/{project_id}/events", h.listEvents)
	r.Get("/api/events/{event_id}", h.getEvent)
	r.Get("/api/projects/{project_id}/stats/trend", h.statsTrend)
	r.Get("/api/projects/{project_id}/stats/levels", h.statsLevels)
	r.Get("/api/projects/{project_id}/stats/top-issues", h.statsTopIssues)
	r.Get("/api/projects/{project_id}/stats/top-releases", h.statsTopReleases)
	r.Get("/api/projects/{project_id}/releases", h.listReleases)
	r.Get("/api/projects/{project_id}/alerts", h.listAlerts)
	r.Post("/api/projects/{project_id}/alerts/webhook", h.createWebhookAlert)
	r.Get("/api/projects/{project_id}/alert-deliveries", h.listAlertDeliveries)
	r.Patch("/api/alerts/{alert_id}/status", h.updateAlertStatus)
	r.Post("/api/alerts/{alert_id}/test", h.testAlert)
}

func (h queryHandler) listIssues(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit := intQuery(r, "limit", 50)
	offset := intQuery(r, "offset", 0)
	items, total, err := h.issues.List(ctx, issue.ListOptions{
		ProjectRef:  chi.URLParam(r, "project_id"),
		Status:      r.URL.Query().Get("status"),
		Level:       r.URL.Query().Get("level"),
		Environment: r.URL.Query().Get("environment"),
		Release:     r.URL.Query().Get("release"),
		Since:       timeQuery(r, "since"),
		Until:       timeQuery(r, "until"),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_issues_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, listResponse(items, total, limit, offset))
}

func (h queryHandler) getIssue(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.issues.Get(ctx, chi.URLParam(r, "issue_id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "issue_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_issue_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h queryHandler) updateIssueStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.issues.UpdateStatus(ctx, chi.URLParam(r, "issue_id"), body.Status, body.Reason)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "issue_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update_issue_status_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h queryHandler) listIssueStatusChanges(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.issues.ListStatusChanges(ctx, chi.URLParam(r, "issue_id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_issue_status_changes_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h queryHandler) listEvents(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	projectID, err := h.projects.ResolveProjectID(ctx, chi.URLParam(r, "project_id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}

	limit := intQuery(r, "limit", 50)
	offset := intQuery(r, "offset", 0)
	items, total, err := h.events.List(ctx, storage.EventQuery{
		ProjectID:   projectID,
		IssueID:     r.URL.Query().Get("issue_id"),
		Level:       r.URL.Query().Get("level"),
		Environment: r.URL.Query().Get("environment"),
		Release:     r.URL.Query().Get("release"),
		Since:       timeQuery(r, "since"),
		Until:       timeQuery(r, "until"),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_events_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, listResponse(items, total, limit, offset))
}

func (h queryHandler) getEvent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.events.Get(ctx, chi.URLParam(r, "event_id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_event_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h queryHandler) statsTrend(w http.ResponseWriter, r *http.Request) {
	h.writeProjectStats(w, r, func(ctx context.Context, projectID string, query storage.StatsQuery) (any, error) {
		return h.stats.Trend(ctx, query)
	})
}

func (h queryHandler) statsLevels(w http.ResponseWriter, r *http.Request) {
	h.writeProjectStats(w, r, func(ctx context.Context, projectID string, query storage.StatsQuery) (any, error) {
		return h.stats.LevelBreakdown(ctx, query)
	})
}

func (h queryHandler) statsTopIssues(w http.ResponseWriter, r *http.Request) {
	h.writeProjectStats(w, r, func(ctx context.Context, projectID string, query storage.StatsQuery) (any, error) {
		return h.stats.TopIssues(ctx, query)
	})
}

func (h queryHandler) statsTopReleases(w http.ResponseWriter, r *http.Request) {
	h.writeProjectStats(w, r, func(ctx context.Context, projectID string, query storage.StatsQuery) (any, error) {
		return h.stats.TopReleases(ctx, query)
	})
}

func (h queryHandler) listReleases(w http.ResponseWriter, r *http.Request) {
	h.writeProjectStats(w, r, func(ctx context.Context, projectID string, query storage.StatsQuery) (any, error) {
		return h.stats.Releases(ctx, query)
	})
}

func (h queryHandler) writeProjectStats(w http.ResponseWriter, r *http.Request, load func(context.Context, string, storage.StatsQuery) (any, error)) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	projectID, err := h.projects.ResolveProjectID(ctx, chi.URLParam(r, "project_id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}

	query := storage.StatsQuery{
		ProjectID:   projectID,
		Environment: r.URL.Query().Get("environment"),
		Release:     r.URL.Query().Get("release"),
		Since:       timeQuery(r, "since"),
		Until:       timeQuery(r, "until"),
		Limit:       intQuery(r, "limit", 10),
	}

	items, err := load(ctx, projectID, query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stats_query_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h queryHandler) listAlerts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	projectID, err := h.projects.ResolveProjectID(ctx, chi.URLParam(r, "project_id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	items, err := h.alerts.ListProjectRules(ctx, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_alerts_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h queryHandler) createWebhookAlert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name            string `json:"name"`
		EventType       string `json:"event_type"`
		WebhookURL      string `json:"webhook_url"`
		MinLevel        string `json:"min_level"`
		ThresholdCount  int    `json:"threshold_count"`
		WindowSeconds   int    `json:"window_seconds"`
		CooldownSeconds int    `json:"cooldown_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}
	if body.Name == "" || body.WebhookURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_alert", "message": "name and webhook_url are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	projectID, err := h.projects.ResolveProjectID(ctx, chi.URLParam(r, "project_id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}

	rule, err := h.alerts.CreateWebhookRule(ctx, projectID, body.Name, body.EventType, body.WebhookURL, body.MinLevel, body.ThresholdCount, body.WindowSeconds, body.CooldownSeconds)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "create_alert_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h queryHandler) updateAlertStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rule, err := h.alerts.UpdateRuleStatus(ctx, chi.URLParam(r, "alert_id"), body.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update_alert_status_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h queryHandler) testAlert(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	rule, err := h.alerts.GetRule(ctx, chi.URLParam(r, "alert_id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_alert_failed", "message": err.Error()})
		return
	}
	if rule.Channel != "webhook" || rule.WebhookURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_alert_test", "message": "only webhook alerts can be tested"})
		return
	}

	payload, err := json.Marshal(map[string]any{
		"type":        "test",
		"project_id":  rule.ProjectID,
		"alert_id":    rule.ID,
		"level":       rule.MinLevel,
		"title":       fmt.Sprintf("Test alert: %s", rule.Name),
		"message":     "This is a Sentry Lite webhook test.",
		"occurred_at": time.Now().UTC(),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_alert_test_failed", "message": err.Error()})
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rule.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_webhook_url", "message": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "sentry-lite-alert-test/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "alert_test_failed", "message": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "alert_test_failed", "message": "webhook returned " + resp.Status})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h queryHandler) listAlertDeliveries(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	projectID, err := h.projects.ResolveProjectID(ctx, chi.URLParam(r, "project_id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	limit := intQuery(r, "limit", 50)
	offset := intQuery(r, "offset", 0)
	items, total, err := h.alerts.ListProjectDeliveries(ctx, projectID, r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_alert_deliveries_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, listResponse(items, total, limit, offset))
}

func intQuery(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < 0 {
		return fallback
	}
	return parsed
}

func timeQuery(r *http.Request, key string) time.Time {
	value := r.URL.Query().Get(key)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func listResponse(items any, total int, limit int, offset int) map[string]any {
	return map[string]any{
		"items": items,
		"page": map[string]int{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	}
}
