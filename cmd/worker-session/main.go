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
	"sentry-lite/internal/session"
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

	writer := storage.NewSessionWriter(deps.ClickHouse)
	pull := worker.PullWorker{
		Name:    cfg.SessionConsumer,
		Subject: cfg.RawSessionSubject,
		Durable: cfg.SessionConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   50,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			items, err := session.ParseRawMessage(msg.Data)
			if err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.SessionConsumer, msg, err)
				return nil
			}
			for _, item := range items {
				if err := writer.InsertSession(ctx, item); err != nil {
					return err
				}
			}
			return nil
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
