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

	normalizer := normalize.NewNormalizer()
	pull := worker.PullWorker{
		Name:    cfg.NormalizeConsumer,
		Subject: cfg.RawEventSubject,
		Durable: cfg.NormalizeConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   20,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			var raw normalize.RawEventMessage
			if err := json.Unmarshal(msg.Data, &raw); err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.NormalizeConsumer, msg, err)
				return nil
			}

			normalized, err := normalizer.Normalize(raw)
			if err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.NormalizeConsumer, msg, err)
				return nil
			}

			body, err := json.Marshal(normalized)
			if err != nil {
				return err
			}
			out := nats.NewMsg(cfg.NormalizedEventSubject)
			out.Data = body
			out.Header.Set("content-type", "application/json")
			out.Header.Set("project-id", normalized.ProjectID)
			out.Header.Set("event-id", normalized.EventID)
			out.Header.Set(nats.MsgIdHdr, normalized.EventID)
			_, err = deps.JetStream.PublishMsg(out)
			return err
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
