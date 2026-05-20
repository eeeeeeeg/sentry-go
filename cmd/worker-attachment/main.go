package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"sentry-lite/internal/attachment"
	"sentry-lite/internal/config"
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

	writer := storage.NewAttachmentWriter(deps.Postgres)
	pull := worker.PullWorker{
		Name:    cfg.AttachmentConsumer,
		Subject: cfg.RawAttachmentSubject,
		Durable: cfg.AttachmentConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   25,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			item, err := attachment.ParseRawMessage(msg.Data)
			if err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.AttachmentConsumer, msg, err)
				return nil
			}
			if item.Filename == "" {
				return nil
			}
			return writer.InsertAttachment(ctx, item)
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
