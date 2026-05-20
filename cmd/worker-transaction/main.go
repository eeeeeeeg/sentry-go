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
	"sentry-lite/internal/storage"
	"sentry-lite/internal/transaction"
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

	writer := storage.NewTransactionWriter(deps.ClickHouse)
	pull := worker.PullWorker{
		Name:    cfg.TransactionConsumer,
		Subject: cfg.RawTransactionSubject,
		Durable: cfg.TransactionConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   50,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			item, spans, err := transaction.ParseRawMessage(msg.Data)
			if err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.TransactionConsumer, msg, err)
				return nil
			}
			if item.EventID == "" {
				return nil
			}
			return writer.InsertTransaction(ctx, item, spans)
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
