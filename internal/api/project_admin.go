package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"sentry-lite/internal/project"
)

type projectAdminHandler struct {
	projects *project.Repository
}

func (h projectAdminHandler) register(r chi.Router) {
	r.Get("/api/projects", h.listProjects)
	r.Post("/api/projects", h.createProject)
	r.Get("/api/projects/{project_id}", h.getProject)
	r.Patch("/api/projects/{project_id}", h.updateProject)
	r.Patch("/api/projects/{project_id}/status", h.updateProjectStatus)
	r.Get("/api/projects/{project_id}/keys", h.listProjectKeys)
	r.Post("/api/projects/{project_id}/keys", h.createProjectKey)
	r.Patch("/api/project-keys/{key_id}", h.updateProjectKey)
	r.Patch("/api/project-keys/{key_id}/status", h.updateProjectKeyStatus)
}

func (h projectAdminHandler) listProjects(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit := intQuery(r, "limit", 20)
	offset := intQuery(r, "offset", 0)
	items, total, err := h.projects.ListProjects(ctx, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_projects_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, listResponse(items, total, limit, offset))
}

func (h projectAdminHandler) getProject(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.projects.GetProject(ctx, chi.URLParam(r, "project_id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_project_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h projectAdminHandler) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrganizationID   string   `json:"organization_id"`
		OrganizationSlug string   `json:"organization_slug"`
		Slug             string   `json:"slug"`
		Name             string   `json:"name"`
		Platform         string   `json:"platform"`
		SampleRate       *float64 `json:"sample_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}
	if body.Slug == "" || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_project", "message": "slug and name are required"})
		return
	}

	organizationRef := body.OrganizationID
	if organizationRef == "" {
		organizationRef = body.OrganizationSlug
	}
	if organizationRef == "" {
		organizationRef = "demo"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sampleRate := 1.0
	if body.SampleRate != nil {
		sampleRate = *body.SampleRate
	}

	item, err := h.projects.CreateProject(ctx, organizationRef, body.Slug, body.Name, body.Platform, sampleRate)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "organization_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "create_project_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h projectAdminHandler) updateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       string   `json:"name"`
		Platform   string   `json:"platform"`
		SampleRate *float64 `json:"sample_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sampleRate := -1.0
	if body.SampleRate != nil {
		sampleRate = *body.SampleRate
	}

	item, err := h.projects.UpdateProject(ctx, chi.URLParam(r, "project_id"), body.Name, body.Platform, sampleRate)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update_project_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h projectAdminHandler) updateProjectStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.projects.UpdateProjectStatus(ctx, chi.URLParam(r, "project_id"), body.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update_project_status_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h projectAdminHandler) listProjectKeys(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit := intQuery(r, "limit", 20)
	offset := intQuery(r, "offset", 0)
	items, total, err := h.projects.ListProjectKeys(ctx, chi.URLParam(r, "project_id"), limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_project_keys_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, listResponse(items, total, limit, offset))
}

func (h projectAdminHandler) createProjectKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name               string `json:"name"`
		RateLimitPerMinute int64  `json:"rate_limit_per_minute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.projects.CreateProjectKey(ctx, chi.URLParam(r, "project_id"), body.Name, body.RateLimitPerMinute)
	if errors.Is(err, project.ErrProjectKeyNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "create_project_key_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h projectAdminHandler) updateProjectKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name               string `json:"name"`
		RateLimitPerMinute int64  `json:"rate_limit_per_minute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.projects.UpdateProjectKey(ctx, chi.URLParam(r, "key_id"), body.Name, body.RateLimitPerMinute)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_key_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update_project_key_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h projectAdminHandler) updateProjectKeyStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json", "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.projects.UpdateProjectKeyStatus(ctx, chi.URLParam(r, "key_id"), body.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project_key_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update_project_key_status_failed", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}
