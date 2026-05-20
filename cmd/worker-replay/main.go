package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"sentry-lite/internal/config"
	"sentry-lite/internal/platform"
	"sentry-lite/internal/replay"
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

	writer := storage.NewReplayWriter(deps.Postgres)
	pull := worker.PullWorker{
		Name:    cfg.ReplayConsumer,
		Subject: cfg.RawReplaySubject,
		Durable: cfg.ReplayConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   25,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			item, err := replay.ParseRawMessage(msg.Data)
			if err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.ReplayConsumer, msg, err)
				return nil
			}
			if item.ReplayID == "" {
				return nil
			}
			return writer.InsertReplayItem(ctx, item)
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
