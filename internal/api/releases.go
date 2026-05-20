package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"sentry-lite/internal/auth"
	"sentry-lite/internal/config"
	"sentry-lite/internal/release"
)

type releaseHandler struct {
	cfg      config.Config
	auth     *auth.Repository
	releases *release.Repository
}

type createReleaseRequest struct {
	Version      string   `json:"version"`
	Projects     []string `json:"projects"`
	Ref          string   `json:"ref"`
	URL          string   `json:"url"`
	DateReleased string   `json:"dateReleased"`
}

type updateReleaseFileRequest struct {
	Name string `json:"name"`
	Dist string `json:"dist"`
}

type createDeployRequest struct {
	Environment  string `json:"environment"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	DateStarted  string `json:"dateStarted"`
	DateFinished string `json:"dateFinished"`
}

func (h releaseHandler) register(r chi.Router) {
	r.Get("/api/0/organizations/{organization_slug}/releases/", h.listOrganizationReleases)
	r.Post("/api/0/organizations/{organization_slug}/releases/", h.createOrganizationRelease)
	r.Get("/api/0/organizations/{organization_slug}/releases/{version}/", h.getOrganizationRelease)
	r.Put("/api/0/organizations/{organization_slug}/releases/{version}/", h.updateOrganizationRelease)
	r.Delete("/api/0/organizations/{organization_slug}/releases/{version}/", h.deleteOrganizationRelease)
	r.Get("/api/0/organizations/{organization_slug}/releases/{version}/deploys/", h.listReleaseDeploys)
	r.Post("/api/0/organizations/{organization_slug}/releases/{version}/deploys/", h.createReleaseDeploy)
	r.Post("/api/0/organizations/{organization_slug}/releases/{version}/files/", h.uploadOrganizationReleaseFile)
	r.Get("/api/0/organizations/{organization_slug}/releases/{version}/files/", h.listOrganizationReleaseFiles)
	r.Get("/api/0/organizations/{organization_slug}/releases/{version}/files/{file_id}/", h.getOrganizationReleaseFile)
	r.Put("/api/0/organizations/{organization_slug}/releases/{version}/files/{file_id}/", h.updateOrganizationReleaseFile)
	r.Delete("/api/0/organizations/{organization_slug}/releases/{version}/files/{file_id}/", h.deleteOrganizationReleaseFile)
	r.Post("/api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/", h.uploadProjectReleaseFile)
	r.Get("/api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/", h.listProjectReleaseFiles)
	r.Get("/api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/{file_id}/", h.getProjectReleaseFile)
	r.Put("/api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/{file_id}/", h.updateProjectReleaseFile)
	r.Delete("/api/0/projects/{organization_slug}/{project_slug}/releases/{version}/files/{file_id}/", h.deleteProjectReleaseFile)
}

func (h releaseHandler) listOrganizationReleases(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "project:releases") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.releases.List(ctx, chi.URLParam(r, "organization_slug"), r.URL.Query().Get("query"), 100)
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h releaseHandler) createOrganizationRelease(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "project:releases") {
		return
	}
	defer r.Body.Close()

	var body createReleaseRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid JSON body"})
		return
	}
	body.Version = strings.TrimSpace(body.Version)
	if body.Version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "version is required"})
		return
	}
	if len(body.Projects) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "projects is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.releases.CreateOrUpdate(ctx, chi.URLParam(r, "organization_slug"), release.CreateReleaseInput{
		Version:      body.Version,
		Projects:     body.Projects,
		Ref:          body.Ref,
		URL:          body.URL,
		DateReleased: body.DateReleased,
	})
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h releaseHandler) getOrganizationRelease(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "org:ci", "project:admin", "project:read", "project:releases", "project:write") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.releases.Get(ctx, chi.URLParam(r, "organization_slug"), releaseVersion(r))
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h releaseHandler) updateOrganizationRelease(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "org:ci", "project:admin", "project:releases", "project:write") {
		return
	}
	defer r.Body.Close()

	var body createReleaseRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid JSON body"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.releases.Update(ctx, chi.URLParam(r, "organization_slug"), releaseVersion(r), release.CreateReleaseInput{
		Ref:          body.Ref,
		URL:          body.URL,
		DateReleased: body.DateReleased,
	})
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h releaseHandler) deleteOrganizationRelease(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "project:admin", "project:releases") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.releases.Delete(ctx, chi.URLParam(r, "organization_slug"), releaseVersion(r)); err != nil {
		h.writeReleaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h releaseHandler) listReleaseDeploys(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "org:ci", "project:admin", "project:read", "project:releases", "project:write") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.releases.ListDeploys(ctx, chi.URLParam(r, "organization_slug"), releaseVersion(r))
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h releaseHandler) createReleaseDeploy(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "org:ci", "project:admin", "project:releases", "project:write") {
		return
	}
	defer r.Body.Close()

	var body createDeployRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid JSON body"})
		return
	}
	body.Environment = strings.TrimSpace(body.Environment)
	if body.Environment == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "environment is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.releases.CreateDeploy(ctx, chi.URLParam(r, "organization_slug"), releaseVersion(r), release.CreateDeployInput{
		Environment:  body.Environment,
		Name:         strings.TrimSpace(body.Name),
		URL:          strings.TrimSpace(body.URL),
		DateStarted:  strings.TrimSpace(body.DateStarted),
		DateFinished: strings.TrimSpace(body.DateFinished),
	})
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h releaseHandler) uploadOrganizationReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.uploadReleaseFile(w, r, "")
}

func (h releaseHandler) uploadProjectReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.uploadReleaseFile(w, r, chi.URLParam(r, "project_slug"))
}

func (h releaseHandler) uploadReleaseFile(w http.ResponseWriter, r *http.Request, projectRef string) {
	if !h.authorize(w, r, "project:releases") {
		return
	}
	version := strings.TrimSpace(releaseVersion(r))
	if version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "version is required"})
		return
	}

	formLimit := h.cfg.MaxReleaseFileBytes + (1 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, formLimit)
	if err := r.ParseMultipartForm(formLimit); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "multipart form-data body is required"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "file is required"})
		return
	}
	defer file.Close()

	content, err := readLimitedReleaseFile(file, h.cfg.MaxReleaseFileBytes)
	if err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"detail": err.Error()})
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" && header != nil {
		name = header.Filename
	}
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "name is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	item, err := h.releases.UploadFile(ctx, release.UploadFileInput{
		OrganizationRef: chi.URLParam(r, "organization_slug"),
		ProjectRef:      projectRef,
		Version:         version,
		Name:            name,
		Dist:            strings.TrimSpace(r.FormValue("dist")),
		Headers:         releaseFileHeaders(r.MultipartForm.Value["header"]),
		Content:         content,
	})
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h releaseHandler) listOrganizationReleaseFiles(w http.ResponseWriter, r *http.Request) {
	h.listReleaseFiles(w, r, "")
}

func (h releaseHandler) listProjectReleaseFiles(w http.ResponseWriter, r *http.Request) {
	h.listReleaseFiles(w, r, chi.URLParam(r, "project_slug"))
}

func (h releaseHandler) listReleaseFiles(w http.ResponseWriter, r *http.Request, projectRef string) {
	if !h.authorize(w, r, "project:releases") {
		return
	}
	version := strings.TrimSpace(releaseVersion(r))
	if version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "version is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.releases.ListFiles(ctx, chi.URLParam(r, "organization_slug"), projectRef, version)
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h releaseHandler) getOrganizationReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.getReleaseFile(w, r, "")
}

func (h releaseHandler) getProjectReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.getReleaseFile(w, r, chi.URLParam(r, "project_slug"))
}

func (h releaseHandler) getReleaseFile(w http.ResponseWriter, r *http.Request, projectRef string) {
	if !h.authorize(w, r, "project:releases") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, content, err := h.releases.GetFile(ctx, chi.URLParam(r, "organization_slug"), projectRef, releaseVersion(r), chi.URLParam(r, "file_id"))
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	if strings.EqualFold(r.URL.Query().Get("download"), "true") {
		for key, value := range item.Headers {
			w.Header().Set(key, value)
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h releaseHandler) updateOrganizationReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.updateReleaseFile(w, r, "")
}

func (h releaseHandler) updateProjectReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.updateReleaseFile(w, r, chi.URLParam(r, "project_slug"))
}

func (h releaseHandler) updateReleaseFile(w http.ResponseWriter, r *http.Request, projectRef string) {
	if !h.authorize(w, r, "project:releases") {
		return
	}
	defer r.Body.Close()

	var body updateReleaseFileRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid JSON body"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.releases.UpdateFile(ctx, chi.URLParam(r, "organization_slug"), projectRef, releaseVersion(r), chi.URLParam(r, "file_id"), strings.TrimSpace(body.Name), strings.TrimSpace(body.Dist))
	if err != nil {
		h.writeReleaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h releaseHandler) deleteOrganizationReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.deleteReleaseFile(w, r, "")
}

func (h releaseHandler) deleteProjectReleaseFile(w http.ResponseWriter, r *http.Request) {
	h.deleteReleaseFile(w, r, chi.URLParam(r, "project_slug"))
}

func (h releaseHandler) deleteReleaseFile(w http.ResponseWriter, r *http.Request, projectRef string) {
	if !h.authorize(w, r, "project:releases") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.releases.DeleteFile(ctx, chi.URLParam(r, "organization_slug"), projectRef, releaseVersion(r), chi.URLParam(r, "file_id")); err != nil {
		h.writeReleaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h releaseHandler) writeReleaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, release.ErrOrganizationNotFound), errors.Is(err, release.ErrProjectNotFound), errors.Is(err, release.ErrReleaseNotFound), errors.Is(err, release.ErrReleaseFileNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
	}
}

func (h releaseHandler) authorize(w http.ResponseWriter, r *http.Request, scopes ...string) bool {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if _, err := h.auth.Authenticate(ctx, r.Header.Get("Authorization"), scopes...); err != nil {
		h.writeAuthError(w, err)
		return false
	}
	return true
}

func (h releaseHandler) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrMissingBearerToken), errors.Is(err, auth.ErrInvalidBearerToken):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": err.Error()})
	case errors.Is(err, auth.ErrMissingScope):
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
	}
}

func releaseVersion(r *http.Request) string {
	version, err := url.PathUnescape(chi.URLParam(r, "version"))
	if err != nil {
		return chi.URLParam(r, "version")
	}
	return version
}

func readLimitedReleaseFile(file io.Reader, maxBytes int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}
	return content, nil
}

func releaseFileHeaders(values []string) map[string]string {
	headers := map[string]string{}
	for _, value := range values {
		key, val, ok := strings.Cut(value, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		headers[key] = strings.TrimSpace(val)
	}
	return headers
}
