package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"sentry-lite/internal/alert"
	"sentry-lite/internal/config"
	"sentry-lite/internal/platform"
	"sentry-lite/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	deps, err := platform.Connect(ctx, cfg)
	if err != nil {
		slog.Error("connect dependencies", "error", err)
		os.Exit(1)
	}
	defer deps.Close()

	repo := alert.NewRepository(deps.Postgres)
	dispatcher := alert.NewDispatcher(repo, deps.Redis, cfg.AlertSuppressWindow)
	pull := worker.PullWorker{
		Name:    cfg.AlertConsumer,
		Subject: cfg.AlertEventSubject,
		Durable: cfg.AlertConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   20,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			var event alert.Event
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.AlertConsumer, msg, err)
				return nil
			}
			return dispatcher.Dispatch(ctx, event)
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
