package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"sentry-lite/internal/alert"
	"sentry-lite/internal/attachment"
	"sentry-lite/internal/auth"
	"sentry-lite/internal/ingest"
	"sentry-lite/internal/issue"
	"sentry-lite/internal/platform"
	"sentry-lite/internal/project"
	"sentry-lite/internal/quota"
	"sentry-lite/internal/release"
	"sentry-lite/internal/storage"
)

func NewRouter(deps *platform.Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logRequests)

	r.Get("/healthz", healthz)
	r.Get("/readyz", readyz(deps))
	r.Get("/metrics", metricsPlaceholder)

	ingestHandler := ingest.NewHandler(
		deps.Config,
		project.NewRepository(deps.Postgres),
		quota.NewLimiter(deps.Redis, deps.Config.RateLimitWindow),
		deps.JetStream,
	)
	ingestHandler.Register(r)

	projectAdminHandler{
		projects: project.NewRepository(deps.Postgres),
	}.register(r)

	releaseHandler{
		cfg:      deps.Config,
		auth:     auth.NewRepository(deps.Postgres),
		releases: release.NewRepository(deps.Postgres),
	}.register(r)

	attachmentHandler{
		auth:        auth.NewRepository(deps.Postgres),
		attachments: attachment.NewRepository(deps.Postgres),
	}.register(r)

	queryHandler{
		projects: project.NewRepository(deps.Postgres),
		issues:   issue.NewRepository(deps.Postgres),
		events:   storage.NewEventQuerier(deps.ClickHouse),
		stats:    storage.NewStatsQuerier(deps.ClickHouse),
		alerts:   alert.NewRepository(deps.Postgres),
	}.register(r)

	return registerStaticUI(r, os.Getenv("UI_DIST_DIR"))
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyz(deps *platform.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		checks := deps.Check(ctx)
		status := http.StatusOK
		for _, check := range checks {
			if !check.OK {
				status = http.StatusServiceUnavailable
				break
			}
		}

		writeJSON(w, status, map[string]any{
			"status": checksStatus(status),
			"checks": checks,
		})
	}
}

func metricsPlaceholder(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte("# metrics will be added in the ingestion milestone\n"))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func checksStatus(status int) string {
	if status == http.StatusOK {
		return "ready"
	}
	return "not_ready"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write json response", "error", err)
	}
}
