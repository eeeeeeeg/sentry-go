package ingest

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"

	"sentry-lite/internal/config"
	"sentry-lite/internal/project"
	"sentry-lite/internal/quota"
)

type Handler struct {
	cfg      config.Config
	projects *project.Repository
	limiter  *quota.Limiter
	js       nats.JetStreamContext
}

type RawEventMessage struct {
	MessageID      string                 `json:"message_id"`
	ReceivedAt     time.Time              `json:"received_at"`
	RemoteIP       string                 `json:"remote_ip"`
	UserAgent      string                 `json:"user_agent"`
	SDKName        string                 `json:"sdk_name,omitempty"`
	SDKVersion     string                 `json:"sdk_version,omitempty"`
	OrganizationID string                 `json:"organization_id"`
	ProjectID      string                 `json:"project_id"`
	ProjectRef     string                 `json:"project_ref"`
	ProjectKeyID   string                 `json:"project_key_id"`
	PublicKey      string                 `json:"public_key"`
	EnvelopeItems  []EnvelopeItemMetadata `json:"envelope_items,omitempty"`
	Payload        json.RawMessage        `json:"payload"`
}

type RawEnvelopeItemMessage struct {
	MessageID      string               `json:"message_id"`
	ReceivedAt     time.Time            `json:"received_at"`
	RemoteIP       string               `json:"remote_ip"`
	UserAgent      string               `json:"user_agent"`
	SDKName        string               `json:"sdk_name,omitempty"`
	SDKVersion     string               `json:"sdk_version,omitempty"`
	OrganizationID string               `json:"organization_id"`
	ProjectID      string               `json:"project_id"`
	ProjectRef     string               `json:"project_ref"`
	ProjectKeyID   string               `json:"project_key_id"`
	PublicKey      string               `json:"public_key"`
	EventID        string               `json:"event_id,omitempty"`
	Item           EnvelopeItemMetadata `json:"item"`
	Payload        json.RawMessage      `json:"payload"`
}

type acceptedResponse struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status"`
}

type categoryRateResult struct {
	Category string
	quota.Result
}

type categoryRateResults map[string]categoryRateResult

func NewHandler(cfg config.Config, projects *project.Repository, limiter *quota.Limiter, js nats.JetStreamContext) *Handler {
	return &Handler{
		cfg:      cfg,
		projects: projects,
		limiter:  limiter,
		js:       js,
	}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/api/{project_id}/envelope", h.HandleEnvelope)
	r.Post("/api/{project_id}/envelope/", h.HandleEnvelope)
	r.Post("/api/{project_id}/store", h.HandleEnvelope)
	r.Post("/api/{project_id}/store/", h.HandleEnvelope)
	r.Options("/api/{project_id}/envelope", h.HandlePreflight)
	r.Options("/api/{project_id}/envelope/", h.HandlePreflight)
	r.Options("/api/{project_id}/store", h.HandlePreflight)
	r.Options("/api/{project_id}/store/", h.HandlePreflight)
}

func (h *Handler) HandlePreflight(w http.ResponseWriter, r *http.Request) {
	writeIngestCORS(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleEnvelope(w http.ResponseWriter, r *http.Request) {
	writeIngestCORS(w)

	projectRef := strings.TrimSpace(chi.URLParam(r, "project_id"))
	if projectRef == "" {
		writeError(w, http.StatusBadRequest, "missing_project_id", "project_id is required")
		return
	}

	publicKey, err := extractPublicKey(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_dsn", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	projectKey, err := h.projects.FindProjectKey(ctx, projectRef, publicKey)
	if err != nil {
		status, code := projectErrorStatus(err)
		writeError(w, status, code, err.Error())
		return
	}

	body, err := readLimitedRequestBody(r, h.cfg.MaxEnvelopeBytes)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", err.Error())
		return
	}
	decoded, err := decodeIngestPayload(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	rateResults, err := h.allowEnvelopeCategories(ctx, projectKey, decoded, clientIP(r))
	if err != nil {
		slog.Error("rate limit check failed", "error", err, "project_id", projectKey.ProjectID)
		writeError(w, http.StatusServiceUnavailable, "rate_limit_unavailable", "rate limit service unavailable")
		return
	}
	writeCategoryRateHeaders(w, rateResults)
	if eventRateLimited(rateResults) {
		h.publishRateLimitOutcomes(projectKey, projectRef, decoded, rateResults, body, r)
		writeSentryRateLimitHeaders(w, rateResults.firstRejected())
		writeError(w, http.StatusTooManyRequests, "rate_limited", "event rate limit exceeded")
		return
	}
	h.publishRateLimitOutcomes(projectKey, projectRef, decoded, rateResults, body, r)
	decoded = decoded.filterAllowedItems(rateResults)
	if !decoded.HasEvent {
		if err := h.publishEnvelopeItems(projectKey, projectRef, decoded, body, r); err != nil {
			slog.Error("publish envelope items", "error", err, "project_id", projectKey.ProjectID)
			writeError(w, http.StatusServiceUnavailable, "queue_unavailable", "event queue unavailable")
			return
		}
		writeJSON(w, http.StatusAccepted, acceptedResponse{
			ID:     decoded.EventID,
			Status: "accepted",
		})
		return
	}

	if err := h.publishEnvelopeItems(projectKey, projectRef, decoded, body, r); err != nil {
		slog.Error("publish envelope items", "error", err, "project_id", projectKey.ProjectID)
		writeError(w, http.StatusServiceUnavailable, "queue_unavailable", "event queue unavailable")
		return
	}

	eventID := decoded.EventID
	raw := RawEventMessage{
		MessageID:      messageID(eventID, body),
		ReceivedAt:     time.Now().UTC(),
		RemoteIP:       clientIP(r),
		UserAgent:      r.UserAgent(),
		SDKName:        firstNonEmpty(decoded.SDKName, r.Header.Get("X-SDK-Name")),
		SDKVersion:     firstNonEmpty(decoded.SDKVersion, r.Header.Get("X-SDK-Version")),
		OrganizationID: projectKey.OrganizationID,
		ProjectID:      projectKey.ProjectID,
		ProjectRef:     projectRef,
		ProjectKeyID:   projectKey.KeyID,
		PublicKey:      projectKey.PublicKey,
		EnvelopeItems:  envelopeItemMetadataList(decoded.Items),
		Payload:        decoded.Payload,
	}

	rawBytes, err := json.Marshal(raw)
	if err != nil {
		slog.Error("marshal raw event", "error", err)
		writeError(w, http.StatusInternalServerError, "marshal_failed", "failed to prepare event")
		return
	}

	msg := nats.NewMsg(h.cfg.RawEventSubject)
	msg.Data = rawBytes
	msg.Header.Set("content-type", "application/json")
	msg.Header.Set("project-id", projectKey.ProjectID)
	msg.Header.Set("project-key-id", projectKey.KeyID)
	if raw.MessageID != "" {
		msg.Header.Set(nats.MsgIdHdr, raw.MessageID)
	}

	if _, err := h.js.PublishMsg(msg); err != nil {
		slog.Error("publish raw event", "error", err, "project_id", projectKey.ProjectID)
		writeError(w, http.StatusServiceUnavailable, "queue_unavailable", "event queue unavailable")
		return
	}

	writeJSON(w, http.StatusAccepted, acceptedResponse{
		ID:     eventID,
		Status: "accepted",
	})
}

func (h *Handler) publishEnvelopeItems(projectKey project.ProjectKey, projectRef string, decoded decodedEnvelope, originalBody []byte, r *http.Request) error {
	receivedAt := time.Now().UTC()
	for index, item := range decoded.Items {
		if item.Type == "event" {
			continue
		}
		raw := RawEnvelopeItemMessage{
			MessageID:      itemMessageID(decoded.EventID, item, index, originalBody),
			ReceivedAt:     receivedAt,
			RemoteIP:       clientIP(r),
			UserAgent:      r.UserAgent(),
			SDKName:        firstNonEmpty(decoded.SDKName, r.Header.Get("X-SDK-Name")),
			SDKVersion:     firstNonEmpty(decoded.SDKVersion, r.Header.Get("X-SDK-Version")),
			OrganizationID: projectKey.OrganizationID,
			ProjectID:      projectKey.ProjectID,
			ProjectRef:     projectRef,
			ProjectKeyID:   projectKey.KeyID,
			PublicKey:      projectKey.PublicKey,
			EventID:        decoded.EventID,
			Item:           item.EnvelopeItemMetadata,
			Payload:        item.Payload,
		}
		body, err := json.Marshal(raw)
		if err != nil {
			return fmt.Errorf("marshal raw envelope item: %w", err)
		}
		msg := nats.NewMsg(h.subjectForEnvelopeItem(item.EnvelopeItemMetadata))
		msg.Data = body
		msg.Header.Set("content-type", "application/json")
		msg.Header.Set("project-id", projectKey.ProjectID)
		msg.Header.Set("project-key-id", projectKey.KeyID)
		msg.Header.Set("item-type", item.Type)
		msg.Header.Set("category", item.Category)
		msg.Header.Set(nats.MsgIdHdr, raw.MessageID)
		if _, err := h.js.PublishMsg(msg); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) publishRateLimitOutcomes(projectKey project.ProjectKey, projectRef string, decoded decodedEnvelope, results categoryRateResults, originalBody []byte, r *http.Request) {
	receivedAt := time.Now().UTC()
	for index, item := range decoded.Items {
		result, ok := results[item.Category]
		if !ok || result.Allowed {
			continue
		}
		raw := map[string]any{
			"message_id":     itemMessageID(decoded.EventID, item, index, originalBody) + "-outcome",
			"received_at":    receivedAt,
			"sdk_name":       firstNonEmpty(decoded.SDKName, r.Header.Get("X-SDK-Name")),
			"sdk_version":    firstNonEmpty(decoded.SDKVersion, r.Header.Get("X-SDK-Version")),
			"project_id":     projectKey.ProjectID,
			"project_key_id": projectKey.KeyID,
			"event_id":       decoded.EventID,
			"category":       item.Category,
			"reason":         "rate_limited",
			"quantity":       1,
			"source":         "server",
		}
		body, err := json.Marshal(raw)
		if err != nil {
			slog.Error("marshal outcome", "error", err, "project_id", projectKey.ProjectID)
			continue
		}
		msg := nats.NewMsg(h.cfg.RawOutcomeSubject)
		msg.Data = body
		msg.Header.Set("content-type", "application/json")
		msg.Header.Set("project-id", projectKey.ProjectID)
		msg.Header.Set("project-key-id", projectKey.KeyID)
		msg.Header.Set("category", item.Category)
		msg.Header.Set("reason", "rate_limited")
		msg.Header.Set(nats.MsgIdHdr, fmt.Sprintf("%s-outcome", itemMessageID(decoded.EventID, item, index, originalBody)))
		if _, err := h.js.PublishMsg(msg); err != nil {
			slog.Error("publish outcome", "error", err, "project_id", projectKey.ProjectID, "project_ref", projectRef)
		}
	}
}

func (h *Handler) subjectForEnvelopeItem(item EnvelopeItemMetadata) string {
	switch item.Category {
	case "transaction":
		return h.cfg.RawTransactionSubject
	case "session":
		return h.cfg.RawSessionSubject
	case "attachment":
		return h.cfg.RawAttachmentSubject
	case "profile":
		return h.cfg.RawProfileSubject
	case "replay":
		return h.cfg.RawReplaySubject
	case "outcome":
		return h.cfg.RawOutcomeSubject
	default:
		return h.cfg.UnsupportedItemSubject
	}
}

func (h *Handler) allowEnvelopeCategories(ctx context.Context, projectKey project.ProjectKey, decoded decodedEnvelope, remoteIP string) (categoryRateResults, error) {
	limit := projectKey.RateLimitPerMinute
	if limit <= 0 {
		limit = h.cfg.DefaultRateLimit
	}
	results := categoryRateResults{}
	for _, category := range envelopeCategories(decoded.Items) {
		rateName := fmt.Sprintf("project:%s:key:%s:ip:%s:category:%s", projectKey.ProjectID, projectKey.KeyID, remoteIP, category)
		rate, err := h.limiter.Allow(ctx, rateName, limit)
		if err != nil {
			return nil, err
		}
		results[category] = categoryRateResult{
			Category: category,
			Result:   rate,
		}
	}
	return results, nil
}

func envelopeCategories(items []EnvelopeItem) []string {
	seen := map[string]struct{}{}
	categories := make([]string, 0, len(items))
	for _, item := range items {
		category := item.Category
		if category == "" {
			category = "default"
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		categories = append(categories, category)
	}
	return categories
}

func eventRateLimited(results categoryRateResults) bool {
	result, ok := results["error"]
	return ok && !result.Allowed
}

func (r categoryRateResults) firstRejected() categoryRateResult {
	for _, result := range r {
		if !result.Allowed {
			return result
		}
	}
	return categoryRateResult{}
}

func readLimitedRequestBody(r *http.Request, maxBytes int64) ([]byte, error) {
	defer r.Body.Close()

	var body io.Reader = r.Body
	var gzipReader *gzip.Reader
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("Content-Encoding")), "gzip") {
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, fmt.Errorf("invalid gzip body: %w", err)
		}
		gzipReader = reader
		defer gzipReader.Close()
		body = gzipReader
	}

	return readLimitedBody(body, maxBytes)
}

func readLimitedBody(body io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(body, maxBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maxBytes {
		return nil, fmt.Errorf("payload exceeds %d bytes", maxBytes)
	}
	return payload, nil
}

func projectErrorStatus(err error) (int, string) {
	switch {
	case errors.Is(err, project.ErrProjectKeyNotFound):
		return http.StatusUnauthorized, "invalid_project_key"
	case errors.Is(err, project.ErrProjectDisabled):
		return http.StatusForbidden, "project_disabled"
	case errors.Is(err, project.ErrProjectKeyDisabled):
		return http.StatusForbidden, "project_key_disabled"
	default:
		return http.StatusInternalServerError, "project_lookup_failed"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeRateHeaders(w http.ResponseWriter, result quota.Result) {
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
	if result.ResetAfter > 0 {
		w.Header().Set("X-RateLimit-Reset-After", fmt.Sprintf("%d", int(result.ResetAfter.Seconds())))
	}
}

func writeCategoryRateHeaders(w http.ResponseWriter, results categoryRateResults) {
	if len(results) == 0 {
		return
	}
	if result, ok := results["error"]; ok {
		writeRateHeaders(w, result.Result)
		return
	}
	for _, result := range results {
		writeRateHeaders(w, result.Result)
		return
	}
}

func writeSentryRateLimitHeaders(w http.ResponseWriter, result categoryRateResult) {
	retryAfter := int(result.ResetAfter.Seconds())
	if retryAfter <= 0 {
		retryAfter = 60
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	category := result.Category
	if category == "" {
		category = "default"
	}
	w.Header().Set("X-Sentry-Rate-Limits", fmt.Sprintf("%d:%s:project:rate_limited", retryAfter, category))
}

func writeIngestCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Encoding, X-Sentry-Auth, X-Sentry-Key, X-SDK-Name, X-SDK-Version")
	w.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset-After, Retry-After, X-Sentry-Rate-Limits")
	w.Header().Set("Access-Control-Max-Age", "600")
}

func extractEventID(body []byte) string {
	var event struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return ""
	}
	return strings.TrimSpace(event.EventID)
}

func messageID(eventID string, body []byte) string {
	if eventID != "" {
		return eventID
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func itemMessageID(eventID string, item EnvelopeItem, index int, body []byte) string {
	if eventID != "" {
		return fmt.Sprintf("%s-%s-%d", eventID, item.Type, index)
	}
	sum := sha256.Sum256(append(body, []byte(fmt.Sprintf(":%s:%d", item.Type, index))...))
	return hex.EncodeToString(sum[:])
}

func envelopeItemMetadataList(items []EnvelopeItem) []EnvelopeItemMetadata {
	if len(items) == 0 {
		return nil
	}
	metadata := make([]EnvelopeItemMetadata, 0, len(items))
	for _, item := range items {
		metadata = append(metadata, item.EnvelopeItemMetadata)
	}
	return metadata
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
			return ip.String()
		}
	}
	if ip := net.ParseIP(r.Header.Get("X-Real-IP")); ip != nil {
		return ip.String()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
