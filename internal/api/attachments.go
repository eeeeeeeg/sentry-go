package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"sentry-lite/internal/attachment"
	"sentry-lite/internal/auth"
)

type attachmentHandler struct {
	auth        *auth.Repository
	attachments *attachment.Repository
}

func (h attachmentHandler) register(r chi.Router) {
	r.Get("/api/0/projects/{organization_slug}/{project_slug}/events/{event_id}/attachments/", h.listEventAttachments)
	r.Get("/api/0/projects/{organization_slug}/{project_slug}/events/{event_id}/attachments/{attachment_id}/", h.getEventAttachment)
	r.Delete("/api/0/projects/{organization_slug}/{project_slug}/events/{event_id}/attachments/{attachment_id}/", h.deleteEventAttachment)
}

func (h attachmentHandler) listEventAttachments(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "event:admin", "event:read", "event:write") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.attachments.ListForEvent(ctx, chi.URLParam(r, "organization_slug"), chi.URLParam(r, "project_slug"), eventIDParam(r))
	if err != nil {
		h.writeAttachmentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h attachmentHandler) getEventAttachment(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "event:admin", "event:read", "event:write") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, content, err := h.attachments.GetForEvent(ctx, chi.URLParam(r, "organization_slug"), chi.URLParam(r, "project_slug"), eventIDParam(r), chi.URLParam(r, "attachment_id"))
	if err != nil {
		h.writeAttachmentError(w, err)
		return
	}
	if strings.EqualFold(r.URL.Query().Get("download"), "true") {
		w.Header().Set("Content-Type", item.ContentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(item.Name, `"`, "")))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h attachmentHandler) deleteEventAttachment(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "event:admin") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.attachments.DeleteForEvent(ctx, chi.URLParam(r, "organization_slug"), chi.URLParam(r, "project_slug"), eventIDParam(r), chi.URLParam(r, "attachment_id")); err != nil {
		h.writeAttachmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h attachmentHandler) authorize(w http.ResponseWriter, r *http.Request, scopes ...string) bool {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if _, err := h.auth.Authenticate(ctx, r.Header.Get("Authorization"), scopes...); err != nil {
		h.writeAuthError(w, err)
		return false
	}
	return true
}

func (h attachmentHandler) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrMissingBearerToken), errors.Is(err, auth.ErrInvalidBearerToken):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": err.Error()})
	case errors.Is(err, auth.ErrMissingScope):
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
	}
}

func (h attachmentHandler) writeAttachmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, attachment.ErrAttachmentNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
	}
}

func eventIDParam(r *http.Request) string {
	return strings.TrimSpace(chi.URLParam(r, "event_id"))
}
