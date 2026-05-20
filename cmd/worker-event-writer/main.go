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

	"sentry-lite/internal/config"
	"sentry-lite/internal/normalize"
	"sentry-lite/internal/platform"
	"sentry-lite/internal/storage"
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

	writer := storage.NewEventWriter(deps.ClickHouse)
	pull := worker.PullWorker{
		Name:    cfg.EventWriterConsumer,
		Subject: cfg.GroupedEventSubject,
		Durable: cfg.EventWriterConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   50,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			var event normalize.NormalizedEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.EventWriterConsumer, msg, err)
				return nil
			}
			if err := writer.InsertEvent(ctx, event); err != nil {
				return err
			}
			return nil
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
