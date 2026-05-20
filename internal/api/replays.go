package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"sentry-lite/internal/auth"
	"sentry-lite/internal/replay"
)

type replayHandler struct {
	auth    *auth.Repository
	replays *replay.Repository
}

func (h replayHandler) register(r chi.Router) {
	r.Get("/api/0/projects/{organization_slug}/{project_slug}/replays/{replay_id}/recording-segments/", h.listRecordingSegments)
	r.Get("/api/0/projects/{organization_slug}/{project_slug}/replays/{replay_id}/recording-segments/{segment_id}/", h.getRecordingSegment)
}

func (h replayHandler) listRecordingSegments(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "project:admin", "project:read", "project:write") {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.replays.ListRecordingSegments(ctx, chi.URLParam(r, "organization_slug"), chi.URLParam(r, "project_slug"), chi.URLParam(r, "replay_id"))
	if err != nil {
		h.writeReplayError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h replayHandler) getRecordingSegment(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r, "project:admin", "project:read", "project:write") {
		return
	}
	segmentID, err := strconv.Atoi(chi.URLParam(r, "segment_id"))
	if err != nil || segmentID < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "segment_id must be a non-negative integer"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, content, err := h.replays.GetRecordingSegment(ctx, chi.URLParam(r, "organization_slug"), chi.URLParam(r, "project_slug"), chi.URLParam(r, "replay_id"), segmentID)
	if err != nil {
		h.writeReplayError(w, err)
		return
	}
	w.Header().Set("Content-Type", item.ContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h replayHandler) authorize(w http.ResponseWriter, r *http.Request, scopes ...string) bool {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if _, err := h.auth.Authenticate(ctx, r.Header.Get("Authorization"), scopes...); err != nil {
		h.writeAuthError(w, err)
		return false
	}
	return true
}

func (h replayHandler) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrMissingBearerToken), errors.Is(err, auth.ErrInvalidBearerToken):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": err.Error()})
	case errors.Is(err, auth.ErrMissingScope):
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
	}
}

func (h replayHandler) writeReplayError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, replay.ErrReplaySegmentNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
	}
}
